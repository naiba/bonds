package vcard

import (
	"bytes"
	"reflect"
	"testing"
)

func TestEncoder(t *testing.T) {
	var b bytes.Buffer
	if err := NewEncoder(&b).Encode(testCard); err != nil {
		t.Fatal("Expected no error when formatting card, got:", err)
	}

	expected := "BEGIN:VCARD\r\nVERSION:4.0\r\nCLIENTPIDMAP:1;urn:uuid:53e374d9-337e-4727-8803-a1e9c14e0556\r\nEMAIL;PID=1.1:jdoe@example.com\r\nFN;PID=1.1:J. Doe\r\nN:Doe;J.;;;\r\nUID:urn:uuid:4fbe8971-0bc3-424c-9c26-36c3e1eff6b1\r\nEND:VCARD\r\n"
	if b.String() != expected {
		t.Errorf("Excpected vcard to be %q, but got %q", expected, b.String())
	}

	card, err := NewDecoder(&b).Decode()
	if err != nil {
		t.Fatal("Expected no error when parsing formatted card, got:", err)
	}

	if !reflect.DeepEqual(card, testCard) {
		t.Errorf("Invalid parsed card: expected \n%+v\n but got \n%+v", testCard, card)
	}
}

func TestFormatLine_withGroup(t *testing.T) {
	l := formatLine("FN", &Field{
		Value: "Akiyama Mio",
		Group: "item1",
	})

	expected := "item1.FN:Akiyama Mio"
	if l != expected {
		t.Errorf("Excpected formatted line with group to be %q, but got %q", expected, l)
	}
}

var testValue = []struct {
	v         string
	formatted string
}{
	{"Hello World!", "Hello World!"},
	{"this is a single value, with a comma encoded", "this is a single value\\, with a comma encoded"},
	{"Mythical Manager\nHyjinx Software Division", "Mythical Manager\\nHyjinx Software Division"},
	{"aa\\\nbb", "aa\\\\\\nbb"},
}

func TestFormatValue(t *testing.T) {
	for _, test := range testValue {
		if formatted := formatValue(test.v); formatted != test.formatted {
			t.Errorf("formatValue(%q): expected %q, got %q", test.v, test.formatted, formatted)
		}
	}
}

func TestEncoderDeterminism(t *testing.T) {
	card := Card{
		"first-key": []*Field{
			{
				Value: "value-a",
				Params: map[string][]string{
					"p-i":   {"s", "ss", "sss"},
					"p-ii":  {"s", "ss", "sss"},
					"p-iii": {"s", "ss", "sss"},
					"p-iv":  {"sss", "ss", "s"},
				},
				Group: "",
			},
			{
				Value: "value-B",
				Params: map[string][]string{
					"p-i":   {"t", "tt", "ttt"},
					"p-ii":  {"t", "tt", "ttt"},
					"p-iii": {"t", "tt", "ttt"},
					"p-iv":  {"ttt", "tt", "t"},
				},
				Group: "",
			},
			{
				Value:  "VALUE-C",
				Params: map[string][]string{},
				Group:  "",
			},
		},
		"second-KEY": []*Field{

			{
				Value: "value-a\\,b",
				Params: map[string][]string{
					"p-i":   {"s", "ss", "sss"},
					"p-ii":  {"s", "ss", "sss"},
					"p-iii": {"s", "ss", "sss"},
					"p-iv":  {"sss", "ss", "s"},
				},
				Group: "G1",
			},
			{
				Value: "value-B\\1,2",
				Params: map[string][]string{
					"p-i":   {"t", "tt", "ttt"},
					"p-ii":  {"t", "tt", "ttt"},
					"p-iii": {"t", "tt", "ttt"},
					"p-iv":  {"ttt", "tt", "t"},
				},
				Group: "GG",
			},
			{
				Value:  "VALUE-C,1,2",
				Params: map[string][]string{},
				Group:  "",
			},
		},
		"THIRD-KEY": []*Field{},
	}
	ToV4(card)
	var b bytes.Buffer
	if err := NewEncoder(&b).Encode(card); err != nil {
		t.Fatal("Expected no error when formatting card, got:", err)
	}
	canonical := b.String()

	for i := 0; i < 100; i++ {
		b.Reset()
		if err := NewEncoder(&b).Encode(card); err != nil {
			t.Fatal("Expected no error when formatting card, got:", err)
		}
		if b.String() != canonical {
			t.Fatalf("Encode: expected canonical %q, got %q", canonical, b.String())
		}
	}
}
