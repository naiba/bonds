package i18n

import (
	"embed"
	"encoding/json"
	"sync"
)

//go:embed en.json zh.json
var localeFS embed.FS

var (
	translations map[string]map[string]string
	once         sync.Once
)

func load() {
	translations = make(map[string]map[string]string)
	for _, lang := range []string{"en", "zh"} {
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
