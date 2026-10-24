package vcard_test

import (
	"io"
	"log"
	"os"
	"strings"

	"github.com/emersion/go-vcard"
)

func ExampleNewDecoder() {
	f, err := os.Open("cards.vcf")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	dec := vcard.NewDecoder(f)
	for {
		card, err := dec.Decode()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		log.Println(card.PreferredValue(vcard.FieldFormattedName))
	}
}

// encoding a vcard can be done as follows

func ExampleNewEncoder() {
	destFile, err := os.Create("cards.vcf")
	if err != nil {
		log.Fatal(err)
	}
	defer destFile.Close()

	// data in order: first name, middle name, last name, telephone number
	contacts := [][4]string{
		{"John", "Webber", "Maxwell", "(+1) 199 8714"},
		{"Donald", "", "Ron", "(+44) 421 8913"},
		{"Eric", "E.", "Peter", "(+37) 221 9903"},
		{"Nelson", "D.", "Patrick", "(+1) 122 8810"},
	}

	var (
		// card is a map of strings to []*vcard.Field objects
		card = make(vcard.Card)

		// destination where the vcard will be encoded to
		enc = vcard.NewEncoder(destFile)
	)

	for _, entry := range contacts {
		// set only the value of a field by using card.SetValue.
		// This does not set parameters
		card.SetValue(vcard.FieldFormattedName, strings.Join(entry[:3], " "))
		card.SetValue(vcard.FieldTelephone, entry[3])

		// set the value of a field and other parameters by using card.Set
		card.Set(vcard.FieldName, &vcard.Field{
			Value: strings.Join(entry[:3], ";"),
			Params: map[string][]string{
				vcard.ParamSortAs: []string{
					entry[0] + " " + entry[2],
				},
			},
		})

		// make the vCard version 4 compliant
		vcard.ToV4(card)
		err := enc.Encode(card)
		if err != nil {
			log.Fatal(err)
		}
	}
}
