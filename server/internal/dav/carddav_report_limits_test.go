package dav

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emersion/go-webdav/carddav"
)

type cardDAVReportReadCounter struct {
	io.Reader
	bytes int64
}

func (r *cardDAVReportReadCounter) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.bytes += int64(n)
	return n, err
}

type cardDAVReportQueryCounter struct {
	carddav.Backend
	calls int
}

func (b *cardDAVReportQueryCounter) QueryAddressObjects(context.Context, string, *carddav.AddressBookQuery) ([]carddav.AddressObject, error) {
	b.calls++
	return nil, nil
}

func TestCardDAVREPORTReadLimits(t *testing.T) {
	const start = `<C:addressbook-query xmlns:C="urn:ietf:params:xml:ns:carddav"><C:filter/>`
	const end = `</C:addressbook-query>`
	for _, tc := range []struct {
		name          string
		contentType   string
		bodySize      int
		contentLength int64
		wantStatus    int
		wantReads     int64
		wantCalls     int
	}{
		{name: "wrong media type", contentType: "text/plain", bodySize: maxCardDAVReportBytes + 1, contentLength: -1, wantStatus: 400},
		{name: "missing media type", bodySize: maxCardDAVReportBytes + 1, contentLength: -1, wantStatus: 400},
		{name: "malformed media type", contentType: "application/xml; charset", bodySize: maxCardDAVReportBytes + 1, contentLength: -1, wantStatus: 400},
		{name: "known oversize", contentType: "application/xml", bodySize: maxCardDAVReportBytes + 1, contentLength: maxCardDAVReportBytes + 1, wantStatus: 413},
		{name: "unknown oversize", contentType: "application/xml", bodySize: 2 * maxCardDAVReportBytes, contentLength: -1, wantStatus: 413, wantReads: maxCardDAVReportBytes + 1},
		{name: "understated length", contentType: "text/xml", bodySize: 2 * maxCardDAVReportBytes, contentLength: 100, wantStatus: 413, wantReads: maxCardDAVReportBytes + 1},
		{name: "exact limit", contentType: "application/xml; charset=utf-8", bodySize: maxCardDAVReportBytes, contentLength: -1, wantStatus: 207, wantReads: maxCardDAVReportBytes, wantCalls: 1},
		{name: "text XML", contentType: "text/xml", bodySize: 256, contentLength: 256, wantStatus: 207, wantReads: 256, wantCalls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := &cardDAVReportReadCounter{Reader: io.MultiReader(strings.NewReader(start), strings.NewReader(strings.Repeat(" ", tc.bodySize-len(start)-len(end))), strings.NewReader(end))}
			req := httptest.NewRequest("REPORT", "/dav/addressbooks/user/book/", body)
			req.ContentLength = tc.contentLength
			req.Header.Set("Content-Type", tc.contentType)
			backend := &cardDAVReportQueryCounter{}
			rec := httptest.NewRecorder()
			newCardDAVHandler(backend).ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus || body.bytes != tc.wantReads || backend.calls != tc.wantCalls {
				t.Fatalf("status=%d read=%d query calls=%d; want %d/%d/%d", rec.Code, body.bytes, backend.calls, tc.wantStatus, tc.wantReads, tc.wantCalls)
			}
			t.Logf("status=%d bytes read=%d query calls=%d", rec.Code, body.bytes, backend.calls)
		})
	}
}

func TestCardDAVREPORTLimitIncludesTrailingBytes(t *testing.T) {
	// The XML decoder stops after the root; an unbounded replay tail would
	// bypass a limit applied only to the initial XML decode.
	const query = `<C:addressbook-query xmlns:C="urn:ietf:params:xml:ns:carddav"><C:filter/></C:addressbook-query>`
	body := &cardDAVReportReadCounter{Reader: io.MultiReader(strings.NewReader(query), strings.NewReader(strings.Repeat(" ", maxCardDAVReportBytes)))}
	req := httptest.NewRequest("REPORT", "/dav/addressbooks/user/book/", body)
	req.Header.Set("Content-Type", "application/xml")
	backend := &cardDAVReportQueryCounter{}
	rec := httptest.NewRecorder()
	newCardDAVHandler(backend).ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge || body.bytes != maxCardDAVReportBytes+1 || backend.calls != 0 {
		t.Fatalf("trailing body bypassed limit: status=%d read=%d query calls=%d", rec.Code, body.bytes, backend.calls)
	}
}
