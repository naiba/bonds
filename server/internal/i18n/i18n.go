package i18n

import (
	"embed"
	"encoding/json"
	"strings"
	"sync"
)

//go:embed en.json zh.json es.json fr.json de.json pt-BR.json pt-PT.json
var localeFS embed.FS

var (
	translations map[string]map[string]string
	once         sync.Once
)

func load() {
	translations = make(map[string]map[string]string)
	for _, lang := range Supported {
		data, err := localeFS.ReadFile(lang + ".json")
		if err != nil {
			panic("i18n: failed to load " + lang + ".json: " + err.Error())
		}
		m := make(map[string]string)
		if err := json.Unmarshal(data, &m); err != nil {
			panic("i18n: failed to parse " + lang + ".json: " + err.Error())
		}
		translations[lang] = m
	}
}

// T translates a message key for the given language.
// Fallback chain: requested lang → "en" → key itself.
func T(lang, key string) string {
	once.Do(load)

	if msgs, ok := translations[lang]; ok {
		if val, ok := msgs[key]; ok {
			return val
		}
	}
	// fallback to English
	if lang != "en" {
		if msgs, ok := translations["en"]; ok {
			if val, ok := msgs[key]; ok {
				return val
			}
		}
	}
	// fallback to key itself
	return key
}

// Tt translates a key and substitutes {{name}}-style placeholders from params.
// Missing params leave their placeholder verbatim — the same drop-in policy
// as i18next on the frontend, so a bug substituting {{user}} with the empty
// string is loud (placeholder visible in the email) instead of silent.
//
// Use this anywhere a translation contains dynamic data (verification links,
// contact names, reminder labels). Plain T is fine for static strings.
func Tt(lang, key string, params map[string]string) string {
	msg := T(lang, key)
	if len(params) == 0 || !strings.Contains(msg, "{{") {
		return msg
	}
	for name, value := range params {
		msg = strings.ReplaceAll(msg, "{{"+name+"}}", value)
	}
	return msg
}
