package dav

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"net/http"

	"github.com/emersion/go-webdav/carddav"
)

type cardDAVQueryDataRequestedKey struct{}

func newCardDAVHandler(backend carddav.Backend) http.Handler {
	handler := &carddav.Handler{Backend: backend, Prefix: "/dav"}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "REPORT" && r.Body != nil {
			// go-webdav decodes absent address-data and <address-data/> into
			// the same zero value. Preserve presence before that information is
			// lost, so metadata queries can release photos after filtering.
			var report struct {
				XMLName xml.Name
				Prop    struct {
					AddressData *struct{} `xml:"urn:ietf:params:xml:ns:carddav address-data"`
				} `xml:"DAV: prop"`
			}
			var consumed bytes.Buffer
			err := xml.NewDecoder(io.TeeReader(r.Body, &consumed)).Decode(&report)
			// Replay exactly what was read, including decoder read-ahead. The
			// DAV handler remains responsible for validation and error responses.
			r.Body = &cardDAVReportBody{Reader: io.MultiReader(&consumed, r.Body), Closer: r.Body}
			if err == nil && report.XMLName == (xml.Name{Space: "urn:ietf:params:xml:ns:carddav", Local: "addressbook-query"}) {
				r = r.WithContext(context.WithValue(r.Context(), cardDAVQueryDataRequestedKey{}, report.Prop.AddressData != nil))
			}
		}
		handler.ServeHTTP(w, r)
	})
}

type cardDAVReportBody struct {
	io.Reader
	io.Closer
}
