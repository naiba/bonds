package dav

import (
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/naiba/bonds/internal/models"
)

type cardDAVSupportedAddressDataResponse struct {
	XMLName   xml.Name `xml:"DAV: response"`
	Href      string   `xml:"DAV: href"`
	Propstats []struct {
		Status string `xml:"DAV: status"`
		Prop   struct {
			SupportedAddressData struct {
				Types []struct {
					ContentType string `xml:"content-type,attr"`
					Version     string `xml:"version,attr"`
				} `xml:"urn:ietf:params:xml:ns:carddav address-data-type"`
			} `xml:"urn:ietf:params:xml:ns:carddav supported-address-data"`
		} `xml:"DAV: prop"`
	} `xml:"DAV: propstat"`
}

type cardDAVSupportedAddressDataReport struct {
	XMLName  xml.Name                              `xml:"DAV: multistatus"`
	Response []cardDAVSupportedAddressDataResponse `xml:"DAV: response"`
}

type cardDAVMultigetErrorResponse struct {
	XMLName   xml.Name `xml:"DAV: response"`
	Href      string   `xml:"DAV: href"`
	Status    string   `xml:"DAV: status"`
	Propstats []struct {
		Status string `xml:"DAV: status"`
		Prop   struct {
			AddressData *struct{} `xml:"urn:ietf:params:xml:ns:carddav address-data"`
		} `xml:"DAV: prop"`
	} `xml:"DAV: propstat"`
	Error *struct {
		SupportedAddressData *struct{} `xml:"urn:ietf:params:xml:ns:carddav supported-address-data"`
	} `xml:"DAV: error"`
}

type cardDAVMultigetErrorReport struct {
	XMLName  xml.Name                       `xml:"DAV: multistatus"`
	Response []cardDAVMultigetErrorResponse `xml:"DAV: response"`
}

func TestDAVSupportedAddressDataAdvertisesOnlyV3(t *testing.T) {
	// Given an authenticated CardDAV address book with a contact.
	e, db := setupDAVHTTPTestWithDB(t)
	userID, email, password := createDAVHTTPTestUser(t, db)
	vaultID, _ := createDAVHTTPTestContact(t, db, userID, "Ada", "Lovelace")
	server := httptest.NewServer(e)
	defer server.Close()

	requestBody := `<?xml version="1.0" encoding="utf-8"?>
<D:propfind xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:carddav">
  <D:prop>
    <C:supported-address-data/>
  </D:prop>
</D:propfind>`
	request, err := http.NewRequest("PROPFIND", server.URL+"/dav/addressbooks/"+userID+"/"+vaultID+"/", strings.NewReader(requestBody))
	if err != nil {
		t.Fatalf("create PROPFIND request: %v", err)
	}
	request.Header.Set("Depth", "0")
	request.Header.Set("Content-Type", "application/xml; charset=utf-8")
	request.SetBasicAuth(email, password)

	// When the client performs a Depth:0 PROPFIND for supported address data.
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("perform PROPFIND request: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read PROPFIND response: %v", err)
	}

	// Then the address book advertises exactly text/vcard version 3.0 and no v4.
	if response.StatusCode != http.StatusMultiStatus {
		t.Fatalf("expected 207 Multi-Status, got %d: %s", response.StatusCode, body)
	}
	var report cardDAVSupportedAddressDataReport
	if err := xml.Unmarshal(body, &report); err != nil {
		t.Fatalf("decode PROPFIND XML: %v", err)
	}
	if len(report.Response) != 1 {
		t.Fatalf("expected one address book response, got %d", len(report.Response))
	}
	addressBook := report.Response[0]
	if addressBook.Href != "/dav/addressbooks/"+userID+"/"+vaultID+"/" {
		t.Fatalf("unexpected address book href %q", addressBook.Href)
	}
	if len(addressBook.Propstats) != 1 {
		t.Fatalf("expected one propstat, got %d", len(addressBook.Propstats))
	}
	if addressBook.Propstats[0].Status != "HTTP/1.1 200 OK" {
		t.Fatalf("expected successful propstat, got %q", addressBook.Propstats[0].Status)
	}
	types := addressBook.Propstats[0].Prop.SupportedAddressData.Types
	if len(types) != 1 {
		t.Fatalf("expected exactly one address-data-type, got %d", len(types))
	}
	if types[0].ContentType != "text/vcard" || types[0].Version != "3.0" {
		t.Fatalf("expected text/vcard version 3.0, got content-type=%q version=%q", types[0].ContentType, types[0].Version)
	}
	for _, addressDataType := range types {
		if addressDataType.ContentType == "text/vcard" && addressDataType.Version == "4.0" {
			t.Fatal("address book must not advertise text/vcard version 4.0")
		}
	}
}

func TestDAVAddressBookMultigetRejectsV4AddressDataRequest(t *testing.T) {
	// Given an authenticated CardDAV address book containing a contact.
	e, db := setupDAVHTTPTestWithDB(t)
	userID, email, password := createDAVHTTPTestUser(t, db)
	vaultID, contactID := createDAVHTTPTestContact(t, db, userID, "Ada", "Lovelace")
	server := httptest.NewServer(e)
	defer server.Close()

	href := "/dav/addressbooks/" + userID + "/" + vaultID + "/" + contactID + ".vcf"
	reportBody := `<?xml version="1.0" encoding="utf-8"?>
<C:addressbook-multiget xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:carddav">
  <D:prop>
    <C:address-data content-type="text/vcard" version="4.0"/>
  </D:prop>
  <D:href>` + href + `</D:href>
</C:addressbook-multiget>`
	request, err := http.NewRequest("REPORT", server.URL+"/dav/addressbooks/"+userID+"/"+vaultID+"/", strings.NewReader(reportBody))
	if err != nil {
		t.Fatalf("create REPORT request: %v", err)
	}
	request.Header.Set("Content-Type", "application/xml; charset=utf-8")
	request.SetBasicAuth(email, password)

	// When the client requests the contact as an unsupported vCard 4.0 representation.
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("perform REPORT request: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read REPORT response: %v", err)
	}

	// Then the target response is a response-level 409 with the CardDAV supported-data error and no address data.
	if response.StatusCode != http.StatusMultiStatus {
		t.Fatalf("expected 207 Multi-Status, got %d: %s", response.StatusCode, body)
	}
	var report cardDAVMultigetErrorReport
	if err := xml.Unmarshal(body, &report); err != nil {
		t.Fatalf("decode REPORT XML: %v", err)
	}
	if len(report.Response) != 1 {
		t.Fatalf("expected one multiget response, got %d", len(report.Response))
	}
	target := report.Response[0]
	if target.Href != href {
		t.Fatalf("expected target href %q, got %q", href, target.Href)
	}
	if target.Status != "HTTP/1.1 409 Conflict" {
		t.Fatalf("expected response-level 409 Conflict, got %q", target.Status)
	}
	if target.Error == nil || target.Error.SupportedAddressData == nil {
		t.Fatal("expected response-level CardDAV supported-address-data error")
	}
	if len(target.Propstats) != 0 {
		t.Fatalf("expected no propstat for a response-level supported-data error, got %d", len(target.Propstats))
	}
}

func TestDAVAddressBookQueryRejectsV4AddressDataRequest(t *testing.T) {
	// Given an authenticated CardDAV address book containing a contact.
	e, db := setupDAVHTTPTestWithDB(t)
	userID, email, password := createDAVHTTPTestUser(t, db)
	vaultID, contactID := createDAVHTTPTestContact(t, db, userID, "Ada", "Lovelace")
	server := httptest.NewServer(e)
	defer server.Close()

	reportBody := `<?xml version="1.0" encoding="utf-8"?>
	<C:addressbook-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:carddav">
	  <D:prop>
	    <C:address-data content-type="text/vcard" version="4.0"/>
	  </D:prop>
	  <C:filter test="anyof">
	    <C:prop-filter name="UID">
	      <C:text-match match-type="equals">` + contactID + `</C:text-match>
	    </C:prop-filter>
	  </C:filter>
	</C:addressbook-query>`
	request, err := http.NewRequest("REPORT", server.URL+"/dav/addressbooks/"+userID+"/"+vaultID+"/", strings.NewReader(reportBody))
	if err != nil {
		t.Fatalf("create REPORT request: %v", err)
	}
	request.Header.Set("Content-Type", "application/xml; charset=utf-8")
	request.SetBasicAuth(email, password)

	// When the client queries for an unsupported vCard 4.0 representation.
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("perform REPORT request: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read REPORT response: %v", err)
	}

	// Then the query is rejected instead of silently returning vCard 3.0 data.
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d: %s", response.StatusCode, body)
	}
	var errorResponse struct {
		XMLName              xml.Name  `xml:"DAV: error"`
		SupportedAddressData *struct{} `xml:"urn:ietf:params:xml:ns:carddav supported-address-data"`
	}
	if err := xml.Unmarshal(body, &errorResponse); err != nil {
		t.Fatalf("decode REPORT error XML: %v", err)
	}
	if errorResponse.SupportedAddressData == nil {
		t.Fatal("expected CardDAV supported-address-data error")
	}
	if strings.Contains(string(body), "BEGIN:VCARD") {
		t.Fatalf("unsupported query must not return address data: %s", body)
	}
}

func TestDAVPutAddressObjectRejectsV4WithoutChangingContact(t *testing.T) {
	// Given an authenticated CardDAV contact stored with its original name.
	e, db := setupDAVHTTPTestWithDB(t)
	userID, email, password := createDAVHTTPTestUser(t, db)
	vaultID, contactID := createDAVHTTPTestContact(t, db, userID, "Ada", "Lovelace")
	server := httptest.NewServer(e)
	defer server.Close()

	cardBody := "BEGIN:VCARD\r\nVERSION:4.0\r\nUID:" + contactID + "\r\nFN:Grace Hopper\r\nN:Hopper;Grace;;;\r\nEND:VCARD\r\n"
	request, err := http.NewRequest(http.MethodPut, server.URL+"/dav/addressbooks/"+userID+"/"+vaultID+"/"+contactID+".vcf", strings.NewReader(cardBody))
	if err != nil {
		t.Fatalf("create PUT request: %v", err)
	}
	request.Header.Set("Content-Type", "text/vcard; charset=utf-8")
	request.SetBasicAuth(email, password)

	// When the client uploads an unsupported vCard 4.0 representation.
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("perform PUT request: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read PUT response: %v", err)
	}

	// Then the server rejects the unsupported format before changing persisted contact data.
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d: %s", response.StatusCode, body)
	}
	var errorResponse struct {
		XMLName              xml.Name  `xml:"DAV: error"`
		SupportedAddressData *struct{} `xml:"urn:ietf:params:xml:ns:carddav supported-address-data"`
	}
	if err := xml.Unmarshal(body, &errorResponse); err != nil {
		t.Fatalf("decode PUT error XML: %v", err)
	}
	if errorResponse.SupportedAddressData == nil {
		t.Fatal("expected CardDAV supported-address-data error")
	}

	var contact models.Contact
	if err := db.First(&contact, "id = ?", contactID).Error; err != nil {
		t.Fatalf("reload contact: %v", err)
	}
	if contact.FirstName == nil || *contact.FirstName != "Ada" {
		t.Fatalf("expected first name to remain Ada, got %v", contact.FirstName)
	}
	if contact.LastName == nil || *contact.LastName != "Lovelace" {
		t.Fatalf("expected last name to remain Lovelace, got %v", contact.LastName)
	}
}
