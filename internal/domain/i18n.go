package domain

// I18n is a per-language lookup. The "en" key is the canonical fallback.
type I18n map[string]string

// Lang returns the translation for lang, falling back to "en", then to "".
// Pure: no logging, no fatal. Callers that need strict validation should
// check IsZero / Has and surface the error themselves.
func (i I18n) Lang(lang string) string {
	if v, ok := i[lang]; ok {
		return v
	}
	if v, ok := i["en"]; ok {
		return v
	}
	return ""
}

// Has reports whether the map has a value for the given language.
func (i I18n) Has(lang string) bool {
	_, ok := i[lang]
	return ok
}
