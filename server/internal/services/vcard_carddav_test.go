package services

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/models"
)

func TestBuildContactCardDAVV3ConvertsCanonicalVCard(t *testing.T) {
	firstName := "Róisín"
	lastName := "Ní"
	nickname := "小林"
	anniversaryType := models.ContactImportantDateType{InternalType: strPtrOrNil("anniversary")}
	photoURL := "https://example.com/photo.jpg"
	phoneType := models.ContactInformationType{Type: strPtrOrNil("phone")}
	emailType := models.ContactInformationType{Type: strPtrOrNil("email")}
	socialType := models.ContactInformationType{Name: strPtrOrNil("Twitter"), Type: strPtrOrNil("social")}
	phoneKind := "mobile"
	emailKind := "work"
	socialKind := "twitter"
	contact := &models.Contact{
		ID:        "contact-200",
		FirstName: &firstName,
		LastName:  &lastName,
		Nickname:  &nickname,
		File:      &models.File{ID: 1, OriginalURL: &photoURL, MimeType: "image/jpeg"},
		ImportantDates: []models.ContactImportantDate{{
			Label:                    "Anniversary",
			Day:                      cardDAVIntPtr(20),
			Month:                    cardDAVIntPtr(6),
			Year:                     cardDAVIntPtr(2015),
			ContactImportantDateType: &anniversaryType,
		}},
		ContactInformations: []models.ContactInformation{
			{Data: "+1-555-0100", Kind: &phoneKind, Pref: true, ContactInformationType: phoneType},
			{Data: "work@example.com", Kind: &emailKind, Pref: true, ContactInformationType: emailType},
			{Data: "https://twitter.com/example", Kind: &socialKind, Pref: true, ContactInformationType: socialType},
		},
	}

	canonical := BuildContactVCard(contact)
	card := BuildContactCardDAVV3(contact)

	if got := card.Value(vcard.FieldVersion); got != "3.0" {
		t.Fatalf("expected VERSION 3.0, got %q", got)
	}
	if got := card.Value(vcard.FieldKind); got != "" {
		t.Fatalf("expected KIND to be absent, got %q", got)
	}
	if got := card.Value(vcard.FieldFormattedName); got != "Róisín Ní" {
		t.Fatalf("expected accented FN, got %q", got)
	}
	name := card.Name()
	if name == nil || name.GivenName != firstName || name.FamilyName != lastName {
		t.Fatalf("expected accented N, got %#v", name)
	}
	if got := card.Value(vcard.FieldNickname); got != nickname {
		t.Fatalf("expected CJK NICKNAME, got %q", got)
	}
	if got := card.Value("X-ANNIVERSARY"); got != "2015-06-20" {
		t.Fatalf("expected X-ANNIVERSARY, got %q", got)
	}
	if got := card.Value(vcard.FieldAnniversary); got != "" {
		t.Fatalf("expected standard ANNIVERSARY to be absent, got %q", got)
	}
	phone := card[vcard.FieldTelephone][0]
	if phone.Params.Get(vcard.ParamPreferred) != "" || !containsString(phone.Params[vcard.ParamType], "pref") {
		t.Fatalf("expected TEL PREF=1 converted to TYPE=pref, got %#v", phone.Params)
	}
	email := card[vcard.FieldEmail][0]
	if email.Params.Get(vcard.ParamPreferred) != "" || !containsString(email.Params[vcard.ParamType], "pref") || !containsString(email.Params[vcard.ParamType], "internet") {
		t.Fatalf("expected EMAIL pref and internet types, got %#v", email.Params)
	}

	socialFields := card[vcardFieldSocialProfile]
	if len(socialFields) != 1 {
		t.Fatalf("expected one social field, got %d", len(socialFields))
	}
	if socialFields[0].Params.Get(vcard.ParamPreferred) != "" || containsString(socialFields[0].Params[vcard.ParamType], "pref") {
		t.Fatalf("expected standalone social PREF removed, got %#v", socialFields[0].Params)
	}
	if impp := card[vcard.FieldIMPP][0]; !containsString(impp.Params[vcard.ParamType], "pref") {
		t.Fatalf("expected IMPP preference represented as TYPE=pref, got %#v", impp.Params)
	}

	photo := card[vcard.FieldPhoto][0]
	if photo.Params.Get(vcard.ParamMediaType) != "" || photo.Params.Get("VALUE") != "uri" {
		t.Fatalf("expected URI PHOTO without MEDIATYPE, got %#v", photo.Params)
	}

	var encoded bytes.Buffer
	if err := vcard.NewEncoder(&encoded).Encode(card); err != nil {
		t.Fatalf("encode CardDAV vCard: %v", err)
	}
	output := encoded.String()
	if !utf8.Valid(encoded.Bytes()) {
		t.Fatal("expected encoded CardDAV vCard to be valid UTF-8")
	}
	for _, forbidden := range []string{"KIND:", "\nANNIVERSARY:", "PREF=1", "MEDIATYPE=", "CHARSET=", "QUOTED-PRINTABLE"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("encoded CardDAV vCard contains forbidden representation %q: %s", forbidden, output)
		}
	}
	if !strings.Contains(output, "VERSION:3.0") || !strings.Contains(output, "Róisín") || !strings.Contains(output, "Ní") || !strings.Contains(output, "小林") {
		t.Fatalf("encoded CardDAV vCard lost UTF-8 semantic values: %s", output)
	}

	if canonical.Value(vcard.FieldVersion) != "4.0" || canonical.Kind() != vcard.KindIndividual {
		t.Fatalf("converter mutated canonical vCard contract: version=%q kind=%q", canonical.Value(vcard.FieldVersion), canonical.Kind())
	}
	if canonical[vcardFieldSocialProfile][0].Params.Get(vcard.ParamPreferred) != "1" {
		t.Fatalf("converter mutated shared canonical social Params: %#v", canonical[vcardFieldSocialProfile][0].Params)
	}
}

func cardDAVIntPtr(value int) *int {
	return &value
}
