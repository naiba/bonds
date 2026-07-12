package services

import (
	"github.com/emersion/go-vcard"
	"github.com/naiba/bonds/internal/models"
)

// BuildContactCardDAVV3 builds the CardDAV-compatible vCard 3.0 representation.
func BuildContactCardDAVV3(contact *models.Contact) vcard.Card {
	card := BuildContactVCard(contact)
	card.SetValue(vcard.FieldVersion, "3.0")
	delete(card, vcard.FieldKind)

	if anniversary := card.Value(vcard.FieldAnniversary); anniversary != "" {
		card.SetValue("X-ANNIVERSARY", anniversary)
		delete(card, vcard.FieldAnniversary)
	}
	for fieldName, fields := range card {
		for _, field := range fields {
			field.Params = cloneVCardParams(field.Params)
			convertCardDAVFieldParams(fieldName, field)
			if fieldName == vcard.FieldEmail && !field.Params.HasType("internet") {
				field.Params.Add(vcard.ParamType, "internet")
			}
		}
	}

	for _, field := range card[vcard.FieldPhoto] {
		delete(field.Params, vcard.ParamMediaType)
		field.Params.Set(vcard.ParamValue, "uri")
	}

	return card
}

func cloneVCardParams(params vcard.Params) vcard.Params {
	clone := make(vcard.Params, len(params))
	for name, values := range params {
		clone[name] = append([]string(nil), values...)
	}
	return clone
}

func convertCardDAVFieldParams(fieldName string, field *vcard.Field) {
	preferred := field.Params.Get(vcard.ParamPreferred)
	if preferred == "" {
		return
	}

	delete(field.Params, vcard.ParamPreferred)
	if preferred == "1" && (fieldName == vcard.FieldTelephone || fieldName == vcard.FieldEmail || fieldName == vcard.FieldIMPP) {
		if !field.Params.HasType("pref") {
			field.Params.Add(vcard.ParamType, "pref")
		}
	}
}
