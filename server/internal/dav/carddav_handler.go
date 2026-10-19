package dav

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"mime"
	"net/http"

	"github.com/emersion/go-webdav/carddav"
)

type cardDAVReportDataRequestedKey struct{}

// REPORT bodies contain selectors and filters, not uploaded cards. Allow a
// base64 filter for the largest supported (10 MiB) photo plus XML overhead,
// while bounding both preparse storage and replay. Large multigets can be split.
const maxCardDAVReportBytes = 16 << 20

func newCardDAVHandler(backend carddav.Backend) http.Handler {
	handler := &carddav.Handler{Backend: backend, Prefix: "/dav"}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "REPORT" && r.Body != nil {
			// Preserve the library's early protocol rejection: invalid media types
			// must not cause us to consume or buffer an attacker-controlled body.
			mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || (mediaType != "application/xml" && mediaType != "text/xml") {
				http.Error(w, "carddav: expected application/xml or text/xml REPORT", http.StatusBadRequest)
				return
			}
			if r.ContentLength > maxCardDAVReportBytes {
				http.Error(w, "carddav: REPORT body exceeds 16 MiB", http.StatusRequestEntityTooLarge)
				return
			}
			// Check actual bytes as well as Content-Length, including chunked
			// bodies and bytes after the XML root. Never replay an unbounded tail.
			body, err := io.ReadAll(io.LimitReader(r.Body, maxCardDAVReportBytes+1))
			if len(body) > maxCardDAVReportBytes {
				http.Error(w, "carddav: REPORT body exceeds 16 MiB", http.StatusRequestEntityTooLarge)
				return
			}
			if err != nil {
				http.Error(w, "carddav: cannot read REPORT body", http.StatusBadRequest)
				return
			}
			// go-webdav decodes absent address-data and <address-data/> into
			// the same zero value. Preserve presence before that information is
			// lost, so metadata queries can release photos after filtering.
			var report struct {
				XMLName xml.Name
				Prop    struct {
					AddressData *struct{} `xml:"urn:ietf:params:xml:ns:carddav address-data"`
				} `xml:"DAV: prop"`
			}
			err = xml.NewDecoder(bytes.NewReader(body)).Decode(&report)
			// Replay the bounded original bytes. The
			// DAV handler remains responsible for validation and error responses.
			r.Body = &cardDAVReportBody{Reader: bytes.NewReader(body), Closer: r.Body}
			if err == nil && report.XMLName.Space == "urn:ietf:params:xml:ns:carddav" &&
				(report.XMLName.Local == "addressbook-query" || report.XMLName.Local == "addressbook-multiget") {
				r = r.WithContext(context.WithValue(r.Context(), cardDAVReportDataRequestedKey{}, report.Prop.AddressData != nil))
			}
		}
		handler.ServeHTTP(w, r)
	})
}

type cardDAVReportBody struct {
	io.Reader
	io.Closer
}
