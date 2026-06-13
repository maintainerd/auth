package security

import (
	"bufio"
	_ "embed"
	"strings"
)

// commonPasswordsRaw embeds the curated common-password blocklist. The list is
// a lowercase, one-per-line set of the most common weak passwords (rockyou /
// SecLists style top entries).
//
// PRODUCTION NOTE: this file can be replaced wholesale with the full SecLists
// "10-million-password-list-top-10000" (or larger) to satisfy OWASP ASVS
// V2.1.7 (>= 10k common passwords). The loader below imposes no size limit, so
// dropping in the larger list requires no code changes.
//
//go:embed common_passwords.txt
var commonPasswordsRaw string

// embeddedCommonPasswords holds the embedded blocklist as a case-insensitive
// lookup set. It is built once at package init from commonPasswordsRaw.
var embeddedCommonPasswords = loadCommonPasswords(commonPasswordsRaw)

func loadCommonPasswords(raw string) map[string]struct{} {
	set := make(map[string]struct{})
	sc := bufio.NewScanner(strings.NewReader(raw))
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		entry := strings.ToLower(strings.TrimSpace(sc.Text()))
		if entry == "" {
			continue
		}
		set[entry] = struct{}{}
	}
	return set
}

// IsEmbeddedCommonPassword reports whether the candidate exactly matches an
// entry in the embedded common-password blocklist. The comparison is
// case-insensitive; matching is on the whole password, not substrings.
func IsEmbeddedCommonPassword(candidate string) bool {
	if len(embeddedCommonPasswords) == 0 {
		return false
	}
	_, ok := embeddedCommonPasswords[strings.ToLower(strings.TrimSpace(candidate))]
	return ok
}
