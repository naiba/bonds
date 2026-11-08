package vcard

import (
	"errors"
	"io"
	"sort"
	"strings"
)

// An Encoder formats cards.
type Encoder struct {
	w io.Writer
}

// NewEncoder creates a new Encoder that writes cards to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w}
}

// Encode formats a card. The card must have a FieldVersion field.
func (enc *Encoder) Encode(c Card) error {
	begin := "BEGIN:VCARD\r\n"
	if _, err := io.WriteString(enc.w, begin); err != nil {
		return err
	}

	version := c.Get(FieldVersion)
	if version == nil {
		return errors.New("vcard: VERSION field missing")
	}
	if _, err := io.WriteString(enc.w, formatLine(FieldVersion, version, version.Value)+"\r\n"); err != nil {
		return err
	}

	var keys []string
	for k := range c {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fields := c[k]
		if strings.EqualFold(k, FieldVersion) {
			continue
		}
		for _, f := range fields {
			if _, err := io.WriteString(enc.w, formatLine(k, f, version.Value)+"\r\n"); err != nil {
				return err
			}
		}
	}

	end := "END:VCARD\r\n"
	_, err := io.WriteString(enc.w, end)
	return err
}

func formatLine(key string, field *Field, version string) string {
	var s string

	if field.Group != "" {
		s += field.Group + "."
	}
	s += key

	var keys []string
	for k := range field.Params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, pk := range keys {
		for _, pv := range field.Params[pk] {
			s += ";" + formatParam(pk, pv)
		}
	}

	if strings.EqualFold(key, FieldOrganization) {
		s += ":" + formatOrganization(field.Organization())
	} else if isURIValue(key, field, version) {
		// URI values are not TEXT: escaping the comma breaks data: PHOTO URIs.
		// Encode line breaks/backslashes as URI octets, never as content lines.
		s += ":" + uriValueFormatter.Replace(field.Value)
	} else {
		s += ":" + formatValue(field.Value)
	}
	return s
}

// Explicit VALUE overrides the version's default type. PHOTO is URI by default
// in v4, but not v3; checking only VALUE=uri corrupts remote photos whose builders
// correctly omit that optional parameter. Keep this independent of data source.
func isURIValue(key string, field *Field, version string) bool {
	if valueType := field.Params.Get(ParamValue); valueType != "" {
		return strings.EqualFold(valueType, "uri")
	}
	switch strings.ToUpper(key) {
	case FieldSource, FieldURL, FieldIMPP:
		return version == "3.0" || version == "4.0"
	case FieldPhoto, FieldGeolocation, FieldLogo, FieldMember, FieldRelated,
		FieldSound, FieldUID, FieldKey, FieldFreeOrBusyURL, FieldCalendarAddressURI, FieldCalendarURI:
		return version == "4.0"
	default:
		return false
	}
}

func formatParam(k, v string) string {
	return k + "=" + formatValue(v)
}

var uriValueFormatter = strings.NewReplacer("\r", "%0D", "\n", "%0A", "\\", "%5C")

var valueFormatter = strings.NewReplacer("\\", "\\\\", "\n", "\\n", ",", "\\,")

func formatValue(v string) string {
	return valueFormatter.Replace(v)
}
