package vcard

import (
	"reflect"
	"testing"
	"time"
)

var testCard = Card{
	"VERSION": []*Field{{Value: "4.0"}},
	"UID":     []*Field{{Value: "urn:uuid:4fbe8971-0bc3-424c-9c26-36c3e1eff6b1"}},
	"FN": []*Field{{
		Value:  "J. Doe",
		Params: Params{"PID": {"1.1"}},
	}},
	"N": []*Field{{Value: "Doe;J.;;;"}},
	"EMAIL": []*Field{{
		Value:  "jdoe@example.com",
		Params: Params{"PID": {"1.1"}},
	}},
	"CLIENTPIDMAP": []*Field{{Value: "1;urn:uuid:53e374d9-337e-4727-8803-a1e9c14e0556"}},
}

var testCardHandmade = Card{
	"VERSION": []*Field{{Value: "4.0"}},
	"N":       []*Field{{Value: "Bloggs;Joe;;;"}},
	"FN":      []*Field{{Value: "Joe Bloggs"}},
	"EMAIL": []*Field{{
		Value:  "me@joebloggs.com",
		Params: Params{"TYPE": {"home"}, "PREF": {"1"}},
	}},
	"TEL": []*Field{{
		Value:  "tel:+44 20 1234 5678",
		Params: Params{"TYPE": {"cell", "home"}, "PREF": {"1"}},
	}},
	"ADR": []*Field{{
		Value:  ";;1 Trafalgar Square;London;;WC2N;United Kingdom",
		Params: Params{"TYPE": {"home"}, "PREF": {"1"}},
	}},
	"URL": []*Field{{
		Value:  "http://joebloggs.com",
		Params: Params{"TYPE": {"home"}, "PREF": {"1"}},
	}},
	"IMPP": []*Field{{
		Value:  "skype:joe.bloggs",
		Params: Params{"TYPE": {"home"}, "PREF": {"1"}},
	}},
	"X-SOCIALPROFILE": []*Field{{
		Value:  "twitter:https://twitter.com/joebloggs",
		Params: Params{"TYPE": {"home"}, "PREF": {"1"}},
	}},
}

var testCardGoogle = Card{
	"VERSION": []*Field{{Value: "3.0"}},
	"N":       []*Field{{Value: "Bloggs;Joe;;;"}},
	"FN":      []*Field{{Value: "Joe Bloggs"}},
	"EMAIL": []*Field{{
		Value:  "me@joebloggs.com",
		Params: Params{"TYPE": {"INTERNET", "HOME"}},
	}},
	"TEL": []*Field{{
		Value:  "+44 20 1234 5678",
		Params: Params{"TYPE": {"CELL"}},
	}},
	"ADR": []*Field{{
		Value:  ";;1 Trafalgar Square;London;;WC2N;United Kingdom",
		Params: Params{"TYPE": {"HOME"}},
	}},
	"URL": []*Field{
		{Value: "http\\://joebloggs.com", Group: "item1"},
		{Value: "http\\://twitter.com/test", Group: "item2"},
	},
	"X-SKYPE": []*Field{{Value: "joe.bloggs"}},
	"X-ABLABEL": []*Field{
		{Value: "_$!<HomePage>!$_", Group: "item1"},
		{Value: "Twitter", Group: "item2"},
	},
}

var testCardApple = Card{
	"VERSION": []*Field{{Value: "3.0"}},
	"N":       []*Field{{Value: "Bloggs;Joe;;;"}},
	"FN":      []*Field{{Value: "Joe Bloggs"}},
	"EMAIL": []*Field{{
		Value:  "me@joebloggs.com",
		Params: Params{"TYPE": {"INTERNET", "HOME", "pref"}},
	}},
	"TEL": []*Field{{
		Value:  "+44 20 1234 5678",
		Params: Params{"TYPE": {"CELL", "VOICE", "pref"}},
	}},
	"ADR": []*Field{{
		Value:  ";;1 Trafalgar Square;London;;WC2N;United Kingdom",
		Params: Params{"TYPE": {"HOME", "pref"}},
	}},
	"URL": []*Field{{
		Value:  "http://joebloggs.com",
		Params: Params{"TYPE": {"pref"}},
		Group:  "item1",
	}},
	"X-ABLABEL": []*Field{
		{Value: "_$!<HomePage>!$_", Group: "item1"},
	},
	"IMPP": []*Field{{
		Value:  "skype:joe.bloggs",
		Params: Params{"X-SERVICE-TYPE": {"Skype"}, "TYPE": {"HOME", "pref"}},
	}},
	"X-SOCIALPROFILE": []*Field{{
		Value:  "https://twitter.com/joebloggs",
		Params: Params{"TYPE": {"twitter"}},
	}},
}

func TestMaybeGet(t *testing.T) {
	l := []string{"a", "b", "c"}

	expected := []string{"a", "b", "c", "", ""}
	for i, exp := range expected {
		if v := maybeGet(l, i); v != exp {
			t.Errorf("maybeGet(l, %v): expected %q but got %q", i, exp, v)
		}
	}
}

func TestCard(t *testing.T) {
	testCardFullName := testCard["FN"][0]
	if field := testCard.Get(FieldFormattedName); testCardFullName != field {
		t.Errorf("Expected card FN field to be %+v but got %+v", testCardFullName, field)
	}
	if v := testCard.Value(FieldFormattedName); v != testCardFullName.Value {
		t.Errorf("Expected card FN field to be %q but got %q", testCardFullName.Value, v)
	}

	if field := testCard.Get("X-IDONTEXIST"); field != nil {
		t.Errorf("Expected card X-IDONTEXIST field to be %+v but got %+v", nil, field)
	}
	if v := testCard.Value("X-IDONTEXIST"); v != "" {
		t.Errorf("Expected card X-IDONTEXIST field value to be %q but got %q", "", v)
	}

	cardMultipleValues := Card{
		"EMAIL": []*Field{
			{Value: "me@example.org", Params: Params{"TYPE": {"home"}}},
			{Value: "me@example.com", Params: Params{"TYPE": {"work"}}},
		},
	}
	expected := []string{"me@example.org", "me@example.com"}
	if values := cardMultipleValues.Values(FieldEmail); !reflect.DeepEqual(expected, values) {
		t.Errorf("Expected card emails to be %+v but got %+v", expected, values)
	}
	if values := cardMultipleValues.Values("X-IDONTEXIST"); values != nil {
		t.Errorf("Expected card X-IDONTEXIST values to be %+v but got %+v", nil, values)
	}
}

func TestCard_AddValue(t *testing.T) {
	card := make(Card)

	name1 := "Akiyama Mio"
	card.AddValue("FN", name1)
	if values := card.Values("FN"); len(values) != 1 || values[0] != name1 {
		t.Errorf("Expected one FN value, got %v", values)
	}

	name2 := "Mio Akiyama"
	card.AddValue("FN", name2)
	if values := card.Values("FN"); len(values) != 2 || values[0] != name1 || values[1] != name2 {
		t.Errorf("Expected two FN values, got %v", values)
	}
}

func TestCard_Preferred(t *testing.T) {
	if pref := testCard.Preferred("X-IDONTEXIST"); pref != nil {
		t.Errorf("Expected card preferred X-IDONTEXIST field to be %+v but got %+v", nil, pref)
	}
	if v := testCard.PreferredValue("X-IDONTEXIST"); v != "" {
		t.Errorf("Expected card preferred X-IDONTEXIST field value to be %q but got %q", "", v)
	}

	cards := []Card{
		{
			"EMAIL": []*Field{
				{Value: "me@example.org", Params: Params{"TYPE": {"home"}}},
				{Value: "me@example.com", Params: Params{"TYPE": {"work"}, "PREF": {"1"}}},
			},
		},
		{
			"EMAIL": []*Field{
				{Value: "me@example.org", Params: Params{"TYPE": {"home"}, "PREF": {"50"}}},
				{Value: "me@example.com", Params: Params{"TYPE": {"work"}, "PREF": {"25"}}},
			},
		},
		// v3.0
		{
			"EMAIL": []*Field{
				{Value: "me@example.org", Params: Params{"TYPE": {"home"}}},
				{Value: "me@example.com", Params: Params{"TYPE": {"work", "pref"}}},
			},
		},
	}

	for _, card := range cards {
		if pref := card.Preferred(FieldEmail); pref != card["EMAIL"][1] {
			t.Errorf("Expected card preferred email to be %+v but got %+v", card["EMAIL"][1], pref)
		}
		if v := card.PreferredValue(FieldEmail); v != "me@example.com" {
			t.Errorf("Expected card preferred email to be %q but got %q", "me@example.com", v)
		}
	}
}

func TestCard_Name(t *testing.T) {
	card := make(Card)
	if name := card.Name(); name != nil {
		t.Errorf("Expected empty card name to be %+v but got %+v", nil, name)
	}
	if names := card.Names(); names != nil {
		t.Errorf("Expected empty card names to be %+v but got %+v", nil, names)
	}

	expectedName := &Name{
		FamilyName: "Doe",
		GivenName:  "J.",
	}
	expectedNames := []*Name{expectedName}
	card.AddName(expectedName)
	if name := card.Name(); !reflect.DeepEqual(expectedName, name) {
		t.Errorf("Expected populated card name to be %+v but got %+v", expectedName, name)
	}
	if names := card.Names(); !reflect.DeepEqual(expectedNames, names) {
		t.Errorf("Expected populated card names to be %+v but got %+v", expectedNames, names)
	}
}

func TestCard_Kind(t *testing.T) {
	card := make(Card)

	if kind := card.Kind(); kind != KindIndividual {
		t.Errorf("Expected kind of empty card to be %q but got %q", KindIndividual, kind)
	}

	card.SetKind(KindOrganization)
	if kind := card.Kind(); kind != KindOrganization {
		t.Errorf("Expected kind of populated card to be %q but got %q", KindOrganization, kind)
	}
}

func TestCard_FormattedNames(t *testing.T) {
	card := make(Card)

	expectedNames := []*Field{{Value: ""}}
	if names := card.FormattedNames(); !reflect.DeepEqual(expectedNames, names) {
		t.Errorf("Expected empty card formatted names to be %+v but got %+v", expectedNames, names)
	}

	expectedNames = []*Field{{Value: "Akiyama Mio"}}
	card.SetValue(FieldFormattedName, expectedNames[0].Value)
	if names := card.FormattedNames(); !reflect.DeepEqual(expectedNames, names) {
		t.Errorf("Expected populated card formatted names to be %+v but got %+v", expectedNames, names)
	}
}

func TestCard_Gender(t *testing.T) {
	card := make(Card)

	var expectedSex Sex
	var expectedIdentity string
	if sex, identity := card.Gender(); sex != expectedSex || identity != expectedIdentity {
		t.Errorf("Expected gender to be (%q %q) but got (%q %q)", expectedSex, expectedIdentity, sex, identity)
	}

	expectedSex = SexFemale
	card.SetGender(expectedSex, expectedIdentity)
	if sex, identity := card.Gender(); sex != expectedSex || identity != expectedIdentity {
		t.Errorf("Expected gender to be (%q %q) but got (%q %q)", expectedSex, expectedIdentity, sex, identity)
	}

	expectedSex = SexOther
	expectedIdentity = "<3"
	card.SetGender(expectedSex, expectedIdentity)
	if sex, identity := card.Gender(); sex != expectedSex || identity != expectedIdentity {
		t.Errorf("Expected gender to be (%q %q) but got (%q %q)", expectedSex, expectedIdentity, sex, identity)
	}
}

func TestCard_Address(t *testing.T) {
	card := make(Card)

	if address := card.Address(); address != nil {
		t.Errorf("Expected empty card address to be nil, got %v", address)
	}
	if addresses := card.Addresses(); addresses != nil {
		t.Errorf("Expected empty card addresses to be nil, got %v", addresses)
	}

	added := &Address{
		StreetAddress: "1 Trafalgar Square",
		Locality:      "London",
		PostalCode:    "WC2N",
		Country:       "United Kingdom",
	}
	card.AddAddress(added)

	equal := func(a, b *Address) bool {
		if (a == nil && b != nil) || (b == nil && a != nil) {
			return false
		}
		a.Field, b.Field = nil, nil
		return reflect.DeepEqual(a, b)
	}

	if address := card.Address(); !equal(added, address) {
		t.Errorf("Expected address to be %+v but got %+v", added, address)
	}
	if addresses := card.Addresses(); len(addresses) != 1 || !equal(added, addresses[0]) {
		t.Errorf("Expected addresses to be %+v, got %+v", []*Address{added}, addresses)
	}
}

func TestCard_Revision(t *testing.T) {
	card := make(Card)

	if rev, err := card.Revision(); err != nil {
		t.Fatal("Expected no error when getting revision of an empty card, got:", err)
	} else if !rev.IsZero() {
		t.Error("Expected a zero time when getting revision of an empty card, got:", rev)
	}

	expected := time.Date(1984, time.November, 4, 0, 0, 0, 0, time.UTC)
	card.SetRevision(expected)
	if rev, err := card.Revision(); err != nil {
		t.Fatal("Expected no error when getting revision of a populated card, got:", err)
	} else if !rev.Equal(rev) {
		t.Errorf("Expected revision to be %v but got %v", expected, rev)
	}
}
