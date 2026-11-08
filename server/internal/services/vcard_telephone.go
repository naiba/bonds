package services

import (
	"net/url"
	"strings"
	"unicode"

	"github.com/emersion/go-vcard"
)

// TEL defaults to text in v4 (phone-number in v3). Only an explicit URI is
// converted: storing its tel: scheme makes Bonds' dial links add it twice.
// Keep the entire subscriber, including escaped parameters, so ext/isub and
// phone-context survive imports, subscriptions, edits and exports without loss.
func vcardInformationValue(field *vcard.Field, group string) (string, error) {
	if group != "phone" {
		return field.Value, nil
	}
	if values, present := field.Params[vcard.ParamValue]; present && (len(values) != 1 || strings.TrimSpace(values[0]) == "") {
		return "", ErrVCardInvalidData
	}
	switch strings.ToLower(field.Params.Get(vcard.ParamValue)) {
	case "", "text", "phone-number":
		return field.Value, nil
	case "uri":
		return telephoneURISubscriber(field.Value)
	default:
		return "", ErrVCardInvalidData
	}
}

func telephoneURISubscriber(value string) (string, error) {
	if len(value) < 4 || !strings.EqualFold(value[:4], "tel:") {
		return "", ErrVCardInvalidData
	}
	subscriber := value[4:]
	decoded, err := url.PathUnescape(subscriber)
	if err != nil || strings.ContainsAny(subscriber, "?\\\"<>") || strings.ContainsFunc(decoded, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
		return "", ErrVCardInvalidData
	}
	parts := strings.Split(subscriber, ";")
	number := parts[0]
	global := strings.HasPrefix(number, "+")
	digits := strings.TrimPrefix(number, "+")
	if digits == "" {
		return "", ErrVCardInvalidData
	}
	hasDigit := false
	for _, c := range digits {
		if c >= '0' && c <= '9' {
			hasDigit = true
			continue
		}
		if strings.ContainsRune("-.()", c) {
			continue
		}
		if !global && strings.ContainsRune("ABCDEFabcdef*#", c) {
			hasDigit = true
			continue
		}
		return "", ErrVCardInvalidData
	}
	if !hasDigit {
		return "", ErrVCardInvalidData
	}
	parameters := map[string]bool{}
	for _, part := range parts[1:] {
		name, parameter, hasValue := strings.Cut(part, "=")
		name = strings.ToLower(name)
		if name == "" || parameters[name] || (hasValue && parameter == "") {
			return "", ErrVCardInvalidData
		}
		for _, c := range name {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return "", ErrVCardInvalidData
			}
		}
		if (name == "ext" || name == "isub" || name == "phone-context") && !hasValue {
			return "", ErrVCardInvalidData
		}
		parameters[name] = true
	}
	if parameters["ext"] && parameters["isub"] || !global && !parameters["phone-context"] {
		return "", ErrVCardInvalidData
	}
	return subscriber, nil
}

// Previous imports could retain the scheme. Match those rows without replacing
// their IDs; a successful subsequent write repairs Data in the same transaction.
func storedVCardInformationValue(value, group string) string {
	if group == "phone" {
		if subscriber, err := telephoneURISubscriber(value); err == nil {
			return subscriber
		}
	}
	return value
}
