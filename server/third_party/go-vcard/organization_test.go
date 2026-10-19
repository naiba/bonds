package vcard

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestOrganizationComponentEscapesRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		wire string
		want []string
	}{
		{`Acme;Sales`, []string{"Acme", "Sales"}},
		{`Acme;`, []string{"Acme", ""}},
		{`;Sales`, []string{"", "Sales"}},
		{`;`, []string{"", ""}},
		{`Acme\;Partners;Sales`, []string{"Acme;Partners", "Sales"}},
		{`Acme\\;Sales`, []string{`Acme\`, "Sales"}},
		{`Acme\, Inc.;Sales`, []string{"Acme, Inc.", "Sales"}},
		{`公司\;研发;销售`, []string{"公司;研发", "销售"}},
		{`Acme\nPartners;Sales`, []string{"Acme\nPartners", "Sales"}},
	} {
		t.Run(tc.wire, func(t *testing.T) {
			card, err := NewDecoder(strings.NewReader("BEGIN:VCARD\r\nVERSION:3.0\r\nORG:" + tc.wire + "\r\nEND:VCARD\r\n")).Decode()
			if err != nil {
				t.Fatal(err)
			}
			if got := card.Get(FieldOrganization).Organization(); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("decoded %q, want %q", got, tc.want)
			}
			for _, field := range []*Field{card.Get(FieldOrganization), NewOrganization(tc.want)} {
				card.Set(FieldOrganization, field)
				var encoded bytes.Buffer
				if err := NewEncoder(&encoded).Encode(card); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(encoded.String(), "ORG:"+tc.wire+"\r\n") {
					t.Fatalf("wrong wire escaping: %q", encoded.String())
				}
				decoded, err := NewDecoder(&encoded).Decode()
				if err != nil {
					t.Fatal(err)
				}
				if got := decoded.Get(FieldOrganization).Organization(); !reflect.DeepEqual(got, tc.want) {
					t.Fatalf("roundtrip %q, want %q", got, tc.want)
				}
			}
		})
	}
}

func TestOrganizationValueMutationInvalidatesComponents(t *testing.T) {
	field := NewOrganization([]string{"Acme;Partners"})
	field.Value = "Other;Sales"
	if got := field.Organization(); !reflect.DeepEqual(got, []string{"Other", "Sales"}) {
		t.Fatalf("stale components: %q", got)
	}
}
