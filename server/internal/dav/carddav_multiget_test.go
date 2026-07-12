package dav

import (
	"encoding/xml"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/emersion/go-vcard"
)

type cardDAVMultigetResponse struct {
	Href     string `xml:"href"`
	Propstat struct {
		Status string `xml:"status"`
		Prop   struct {
			ETag        string `xml:"getetag"`
			ContentType string `xml:"getcontenttype"`
			AddressData string `xml:"address-data"`
		} `xml:"prop"`
	} `xml:"propstat"`
}

type cardDAVMultigetReport struct {
	XMLName   xml.Name                  `xml:"DAV: multistatus"`
	Responses []cardDAVMultigetResponse `xml:"response"`
}

func TestDAVAddressBookMultigetReturnsUTF8VCard3(t *testing.T) {
	// Given an authenticated DAV server with a non-ASCII contact.
	e, db := setupDAVHTTPTestWithDB(t)
	userID, email, password := createDAVHTTPTestUser(t, db)
	vaultID, contactID := createDAVHTTPTestContact(t, db, userID, "Róisín", "Ní")
	server := httptest.NewServer(e)
	defer server.Close()

	href := "/dav/addressbooks/" + userID + "/" + vaultID + "/" + contactID + ".vcf"
	report := `<?xml version="1.0" encoding="utf-8"?>
<C:addressbook-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:carddav">
  <D:prop>
    <D:getetag/>
    <D:getcontenttype/>
    <C:address-data content-type="text/vcard" version="3.0"/>
  </D:prop>
  <D:href>` + href + `</D:href>
</C:addressbook-multiget>`

	request, err := http.NewRequest(http.MethodPost, server.URL+"/dav/addressbooks/"+userID+"/"+vaultID+"/", strings.NewReader(report))
	if err != nil {
		t.Fatalf("create REPORT request: %v", err)
	}
	request.Method = "REPORT"
	request.Header.Set("Content-Type", "application/xml; charset=utf-8")
	request.SetBasicAuth(email, password)

	// When a real HTTP client performs an addressbook-multiget REPORT.
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("perform REPORT request: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read REPORT response: %v", err)
	}

	// Then the multiget response exposes an iOS-compatible UTF-8 vCard 3.0.
	if response.StatusCode != http.StatusMultiStatus {
		t.Fatalf("expected 207 Multi-Status, got %d: %s", response.StatusCode, body)
	}
	if !utf8.Valid(body) {
		t.Fatal("expected REPORT response body to be valid UTF-8")
	}
	mediaType, params, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse REPORT Content-Type: %v", err)
	}
	if mediaType != "text/xml" && mediaType != "application/xml" {
		t.Fatalf("expected XML media type, got %q", mediaType)
	}
	if params["charset"] != "utf-8" {
		t.Fatalf("expected XML charset=utf-8, got %q", params["charset"])
	}
	if strings.Contains(string(body), "RÃ³isÃ­n") || strings.Contains(string(body), "NÃ­") {
		t.Fatalf("REPORT response contains mojibake: %s", body)
	}

	var reportResponse cardDAVMultigetReport
	if err := xml.Unmarshal(body, &reportResponse); err != nil {
		t.Fatalf("decode REPORT XML: %v", err)
	}
	if len(reportResponse.Responses) != 1 {
		t.Fatalf("expected one multiget response, got %d", len(reportResponse.Responses))
	}
	addressObject := reportResponse.Responses[0]
	if addressObject.Href != href {
		t.Fatalf("expected href %q, got %q", href, addressObject.Href)
	}
	if addressObject.Propstat.Status != "HTTP/1.1 200 OK" {
		t.Fatalf("expected successful propstat, got %q", addressObject.Propstat.Status)
	}
	if addressObject.Propstat.Prop.ETag == "" {
		t.Fatal("expected non-empty ETag")
	}
	if addressObject.Propstat.Prop.ContentType != "text/vcard" {
		t.Fatalf("expected getcontenttype text/vcard, got %q", addressObject.Propstat.Prop.ContentType)
	}
	if addressObject.Propstat.Prop.AddressData == "" {
		t.Fatal("expected non-empty address-data")
	}

	card, err := vcard.NewDecoder(strings.NewReader(addressObject.Propstat.Prop.AddressData)).Decode()
	if err != nil {
		t.Fatalf("decode address-data vCard: %v", err)
	}
	if got := card.Value(vcard.FieldVersion); got != "3.0" {
		t.Fatalf("expected address-data VERSION 3.0, got %q", got)
	}
	if got := card.Value(vcard.FieldFormattedName); got != "Róisín Ní" {
		t.Fatalf("expected address-data FN Róisín Ní, got %q", got)
	}
	name := card.Name()
	if name == nil {
		t.Fatal("expected address-data N component")
	}
	if name.GivenName != "Róisín" || name.FamilyName != "Ní" {
		t.Fatalf("expected address-data N Róisín Ní, got given=%q family=%q", name.GivenName, name.FamilyName)
	}
}
