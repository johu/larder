// Package i18n resolves the language of a request and looks up localized
// messages.
package i18n

import (
	"strconv"
	"strings"
)

// Lang is a supported language, written as its primary language subtag.
type Lang string

const (
	EN Lang = "en"
	DE Lang = "de"
	PT Lang = "pt" // European Portuguese wording; pt-BR and pt-PT both map here
)

// Fallback is the language used when a message is missing in the requested
// language.
const Fallback = EN

// Supported lists all supported languages. Adding a language means adding
// it here and adding locales/<lang>.json.
var Supported = []Lang{EN, DE, PT}

// Match maps a single BCP 47 language tag such as "de", "DE" or "pt-BR" to a
// supported language, using the primary subtag only and ignoring case.
// It reports false for unsupported or empty tags and for "*".
func Match(tag string) (Lang, bool) {
	primary := strings.ToLower(strings.Split(strings.TrimSpace(tag), "-")[0])

	for _, l := range Supported {
		if Lang(primary) == l {
			return l, true
		}
	}

	return "", false
}

// FromAcceptLanguage returns the best supported language for an
// Accept-Language header value.
//
// Rules (see i18n_test.go):
//   - entries are comma separated, each is "<tag>" or "<tag>;q=<weight>"
//   - a missing weight means 1; a weight of 0 means "not acceptable"
//   - the highest weight wins; on equal weights the entry listed first wins
//   - unsupported tags and "*" are skipped
//   - malformed entries are skipped, they must not fail the whole header
//
// It reports false when no entry matches a supported language.
func FromAcceptLanguage(header string) (Lang, bool) {
	var best Lang
	bestQ := -1.0

	for entry := range strings.SplitSeq(header, ",") {
		parts := strings.Split(strings.TrimSpace(entry), ";")
		if len(parts) > 2 {
			continue
		}

		tag := strings.TrimSpace(parts[0])
		if tag == "" || tag == "*" {
			continue
		}

		q := 1.0
		if len(parts) == 2 {
			param := strings.TrimSpace(strings.ToLower(parts[1]))
			if !strings.HasPrefix(param, "q=") {
				continue
			}

			var err error
			q, err = strconv.ParseFloat(strings.TrimPrefix(param, "q="), 64)
			if err != nil || !(q >= 0 && q <= 1) {
				continue
			}
		}

		if q == 0 || q <= bestQ {
			continue
		}

		lang, ok := Match(tag)
		if !ok {
			continue
		}

		best = lang
		bestQ = q
	}

	return best, best != ""
}

// Resolve picks the language for a request: the ?lang= value first, then the
// Accept-Language header, then def. A value that cannot be matched is
// ignored and the next source is tried; it is never an error.
func Resolve(query, acceptLanguage string, def Lang) Lang {
	lang, ok := Match(query)
	if ok {
		return lang
	}

	accLang, ok := FromAcceptLanguage(acceptLanguage)
	if ok {
		return accLang
	}

	return def
}
