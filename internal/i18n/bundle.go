package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
)

//go:embed locales/*.json
var localesFS embed.FS

// Bundle holds the message catalogs of all supported languages.
type Bundle struct {
	// catalogs maps a language to its messages (key -> message). The
	// parity test reads it directly.
	catalogs map[Lang]map[string]string
}

// NewBundle reads <lang>.json for every language in Supported from the root
// of fsys. Each file is a flat JSON object that maps a message key to the
// message. It returns an error if a file is missing or is not valid JSON.
func NewBundle(fsys fs.FS) (*Bundle, error) {
	b := &Bundle{
		catalogs: make(map[Lang]map[string]string, len(Supported)),
	}

	for _, lang := range Supported {
		filename := string(lang) + ".json"

		data, err := fs.ReadFile(fsys, filename)
		if err != nil {
			return nil, fmt.Errorf("i18n: read catalog %q: %w", filename, err)
		}

		var messages map[string]string
		if err := json.Unmarshal(data, &messages); err != nil {
			return nil, fmt.Errorf("i18n: parse catalog %q: %w", filename, err)
		}

		if messages == nil {
			return nil, fmt.Errorf("i18n: catalog %q must be a JSON object", filename)
		}

		b.catalogs[lang] = messages
	}

	return b, nil
}

// Load returns the Bundle built from the catalogs embedded in the binary
// (the locales directory).
func Load() (*Bundle, error) {
	fsys, err := fs.Sub(localesFS, "locales")
	if err != nil {
		return nil, fmt.Errorf("i18n: access embedded locales: %w", err)
	}

	return NewBundle(fsys)
}

// Message returns the message for key in lang. If lang has no such key it
// uses Fallback; if that has none either it returns key itself, so a missing
// translation is visible but never an empty string.
func (b *Bundle) Message(lang Lang, key string) string {
	if message := b.lookup(lang, key); message != "" {
		return message
	}

	if message := b.lookup(Fallback, key); message != "" {
		return message
	}

	return key
}

func (b *Bundle) lookup(lang Lang, key string) string {
	if messages, ok := b.catalogs[lang]; ok {
		if message, ok := messages[key]; ok && message != "" {
			return message
		}
	}

	return ""
}
