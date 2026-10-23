package vcard

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// RFC
var testCardString = `BEGIN:VCARD
VERSION:4.0
UID:urn:uuid:4fbe8971-0bc3-424c-9c26-36c3e1eff6b1
FN;PID=1.1:J. Doe
N:Doe;J.;;;
EMAIL;PID=1.1:jdoe@example.com
CLIENTPIDMAP:1;urn:uuid:53e374d9-337e-4727-8803-a1e9c14e0556
END:VCARD`

var testCardHandmadeString = `BEGIN:VCARD
VERSION:4.0
N:Bloggs;Joe;;;
FN:Joe Bloggs
EMAIL;TYPE=home;PREF=1:me@joebloggs.com
TEL;TYPE="cell,home";PREF=1:tel:+44 20 1234 5678
ADR;TYPE=home;PREF=1:;;1 Trafalgar Square;London;;WC2N;United Kingdom
URL;TYPE=home;PREF=1:http://joebloggs.com
IMPP;TYPE=home;PREF=1:skype:joe.bloggs
X-SOCIALPROFILE;TYPE=home;PREF=1:twitter:https://twitter.com/joebloggs
END:VCARD`

// Google Contacts (15 November 2012)
var testCardGoogleString = `BEGIN:VCARD
VERSION:3.0
N:Bloggs;Joe;;;
FN:Joe Bloggs
EMAIL;TYPE=INTERNET;TYPE=HOME:me@joebloggs.com
TEL;TYPE=CELL:+44 20 1234 5678
ADR;TYPE=HOME:;;1 Trafalgar Square;London;;WC2N;United Kingdom
item1.URL:http\://joebloggs.com
item1.X-ABLabel:_$!<HomePage>!$_
X-SKYPE:joe.bloggs
item2.URL:http\://twitter.com/test
item2.X-ABLabel:Twitter
END:VCARD`

// Apple Contacts (version 7.1)
var testCardAppleString = `BEGIN:VCARD
VERSION:3.0
N:Bloggs;Joe;;;
FN:Joe Bloggs
EMAIL;type=INTERNET;type=HOME;type=pref:me@joebloggs.com
TEL;type=CELL;type=VOICE;type=pref:+44 20 1234 5678
ADR;type=HOME;type=pref:;;1 Trafalgar Square;London;;WC2N;United Kingdom
item1.URL;type=pref:http://joebloggs.com
item1.X-ABLabel:_$!<HomePage>!$_
IMPP;X-SERVICE-TYPE=Skype;type=HOME;type=pref:skype:joe.bloggs
X-SOCIALPROFILE;type=twitter:https://twitter.com/joebloggs
END:VCARD`

var testCardLineFoldingString = `BEGIN:VCARD
VERSION:4.0

NOTE:This is a long description
  that exists o
 n a long line.

END:VCARD
`

var testCardLineFolding = Card{
	"VERSION": {{Value: "4.0"}},
	"NOTE":    {{Value: "This is a long description that exists on a long line."}},
}

var decoderTests = []struct {
	s    string
	card Card
}{
	{testCardString, testCard},
	{testCardHandmadeString, testCardHandmade},
	{testCardGoogleString, testCardGoogle},
	{testCardAppleString, testCardApple},
	{testCardLineFoldingString, testCardLineFolding},
}

func TestDecoder(t *testing.T) {
	for _, test := range decoderTests {
		r := strings.NewReader(test.s)
		dec := NewDecoder(r)
		card, err := dec.Decode()
		if err != nil {
			t.Fatal("Expected no error when decoding card, got:", err)
		}
		if !reflect.DeepEqual(card, test.card) {
			t.Errorf("Invalid parsed card: expected \n%+v\n but got \n%+v", test.card, card)
			for k, fields := range test.card {
				t.Log(k, reflect.DeepEqual(fields, card[k]), fields[0], card[k][0])
			}
		}
	}
}

const testInvalidBegin = `BEGIN:INVALID
END:VCARD`

const testInvalidEnd = `BEGIN:VCARD
END:INVALID`

const testInvalidNoBegin = `VERSION:4.0
END:VCARD`

const testInvalidNoEnd = `BEGIN:VCARD
VERSION:4.0`

var decoderInvalidTests = []string{
	testInvalidBegin,
	testInvalidEnd,
	testInvalidNoBegin,
	testInvalidNoEnd,
}

func TestDecoder_invalid(t *testing.T) {
	for _, test := range decoderInvalidTests {
		r := strings.NewReader(test)
		dec := NewDecoder(r)
		if _, err := dec.Decode(); err == nil {
			t.Fatalf("Expected error when decoding invalid card:\n%v", test)
		}
	}
}

// A single logical line may be split into an unbounded number of folded
// continuation lines. Accumulating them must not take time quadratic in the
// input size, otherwise a modestly sized card can pin a CPU for a long time.
func TestDecoder_manyFoldedLines(t *testing.T) {
	const n = 500000
	var b strings.Builder
	b.WriteString("BEGIN:VCARD\nNOTE:")
	for i := 0; i < n; i++ {
		b.WriteString("\n a")
	}
	b.WriteString("\nEND:VCARD\n")

	done := make(chan struct{})
	var card Card
	var err error
	go func() {
		dec := NewDecoder(strings.NewReader(b.String()))
		card, err = dec.Decode()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("decoding a heavily folded card did not finish in time")
	}

	if err != nil {
		t.Fatal("Expected no error when decoding folded card, got:", err)
	}
	if got := card.Value("NOTE"); len(got) != n {
		t.Errorf("unexpected folded NOTE length: got %d, want %d", len(got), n)
	}
}

func TestParseLine_escaped(t *testing.T) {
	l := "NOTE:Mythical Manager\\nHyjinx Software Division\\nBabsCo\\, Inc.\\n"
	expectedKey := "NOTE"
	expectedValue := "Mythical Manager\nHyjinx Software Division\nBabsCo, Inc.\n"

	if key, field, err := parseLine(l); err != nil {
		t.Fatal("Expected no error while parsing line, got:", err)
	} else if key != expectedKey || field.Value != expectedValue {
		t.Errorf("parseLine(%q): expected (%q, %q), got (%q, %q)", l, expectedKey, expectedValue, key, field.Value)
	}
}
