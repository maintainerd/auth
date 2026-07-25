package security

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// PasswordPolicy defines the complexity and lifecycle rules applied to a user's password.
// All fields are optional overrides — unset fields fall back to DefaultPasswordPolicy().
type PasswordPolicy struct {
	MinLength        int  `json:"min_length"`
	MaxLength        int  `json:"max_length"`
	RequireUpper     bool `json:"require_upper"`
	RequireLower     bool `json:"require_lower"`
	RequireDigit     bool `json:"require_digit"`
	RequireSpecial   bool `json:"require_special"`
	BlocklistEnabled bool `json:"blocklist_enabled"`
	// HistoryCount is the number of previous hashes to check for reuse. 0 disables history.
	HistoryCount int `json:"history_count"`
	// ExpiryDays forces password change after N days. 0 disables expiry.
	ExpiryDays int `json:"expiry_days"`
	// CheckHIBP checks k-anonymity against HaveIBeenPwned on password change.
	CheckHIBP bool `json:"check_hibp"`
	// MinStrengthScore is the minimum guessability score, 0-4, as produced by
	// PasswordStrengthScore. 0 disables the check. The score measures how many
	// guesses the password would survive — it does NOT reward character
	// classes, so raising it never silently reimposes composition rules a
	// tenant has turned off. See strength.go for the model.
	MinStrengthScore int `json:"min_strength_score"`
	// HashAlgorithm selects the password hashing function.
	HashAlgorithm string `json:"hash_algorithm"`
	// TempPasswordValidityHours is the lifetime of admin-generated temp passwords.
	TempPasswordValidityHours int `json:"temporary_password_validity_hours"`
}

// DefaultPasswordPolicy returns the system-wide baseline that applies when no
// tenant-specific policy is configured. MinStrengthScore and CheckHIBP default
// to 0/false here; they are enabled by tenant-level security_settings.
func DefaultPasswordPolicy() PasswordPolicy {
	return PasswordPolicy{
		MinLength:        8,
		MaxLength:        128,
		RequireUpper:     true,
		RequireLower:     true,
		RequireDigit:     true,
		RequireSpecial:   true,
		BlocklistEnabled: true,
		HashAlgorithm:    "argon2id",
	}
}

// MergePasswordPolicy parses raw JSON (e.g. SecuritySetting.PasswordConfig) and
// merges it over the defaults. Fields absent in the JSON keep their default values.
func MergePasswordPolicy(raw []byte) PasswordPolicy {
	policy := DefaultPasswordPolicy()
	if len(raw) == 0 {
		return policy
	}
	s := strings.TrimSpace(string(raw))
	if s == "{}" || s == "null" || s == "" {
		return policy
	}
	var override struct {
		MinLength                      *int    `json:"min_length"`
		MaxLength                      *int    `json:"max_length"`
		RequireUpper                   *bool   `json:"require_upper"`
		RequireUppercase               *bool   `json:"require_uppercase"`
		RequireLower                   *bool   `json:"require_lower"`
		RequireLowercase               *bool   `json:"require_lowercase"`
		RequireDigit                   *bool   `json:"require_digit"`
		RequireNumber                  *bool   `json:"require_number"`
		RequireSpecial                 *bool   `json:"require_special"`
		RequireSymbol                  *bool   `json:"require_symbol"`
		BlocklistEnabled               *bool   `json:"blocklist_enabled"`
		RejectCommonPasswords          *bool   `json:"reject_common_passwords"`
		HistoryCount                   *int    `json:"history_count"`
		PasswordHistoryCount           *int    `json:"password_history_count"`
		ExpiryDays                     *int    `json:"expiry_days"`
		MaxAgeDays                     *int    `json:"max_age_days"`
		CheckHIBP                      *bool   `json:"check_hibp"`
		MinStrengthScore               *int    `json:"min_strength_score"`
		HashAlgorithm                  *string `json:"hash_algorithm"`
		TemporaryPasswordValidityHours *int    `json:"temporary_password_validity_hours"`
	}
	if err := json.Unmarshal(raw, &override); err != nil {
		return policy
	}
	if override.MinLength != nil {
		policy.MinLength = *override.MinLength
	}
	if override.MaxLength != nil {
		policy.MaxLength = *override.MaxLength
	}
	if override.RequireUpper != nil {
		policy.RequireUpper = *override.RequireUpper
	}
	if override.RequireUppercase != nil {
		policy.RequireUpper = *override.RequireUppercase
	}
	if override.RequireLower != nil {
		policy.RequireLower = *override.RequireLower
	}
	if override.RequireLowercase != nil {
		policy.RequireLower = *override.RequireLowercase
	}
	if override.RequireDigit != nil {
		policy.RequireDigit = *override.RequireDigit
	}
	if override.RequireNumber != nil {
		policy.RequireDigit = *override.RequireNumber
	}
	if override.RequireSpecial != nil {
		policy.RequireSpecial = *override.RequireSpecial
	}
	if override.RequireSymbol != nil {
		policy.RequireSpecial = *override.RequireSymbol
	}
	if override.BlocklistEnabled != nil {
		policy.BlocklistEnabled = *override.BlocklistEnabled
	}
	if override.RejectCommonPasswords != nil {
		policy.BlocklistEnabled = *override.RejectCommonPasswords
	}
	if override.HistoryCount != nil {
		policy.HistoryCount = *override.HistoryCount
	}
	if override.PasswordHistoryCount != nil {
		policy.HistoryCount = *override.PasswordHistoryCount
	}
	if override.ExpiryDays != nil {
		policy.ExpiryDays = *override.ExpiryDays
	}
	if override.MaxAgeDays != nil {
		policy.ExpiryDays = *override.MaxAgeDays
	}
	if override.CheckHIBP != nil {
		policy.CheckHIBP = *override.CheckHIBP
	}
	if override.MinStrengthScore != nil {
		policy.MinStrengthScore = *override.MinStrengthScore
	}
	if override.HashAlgorithm != nil {
		policy.HashAlgorithm = *override.HashAlgorithm
	}
	if override.TemporaryPasswordValidityHours != nil {
		policy.TempPasswordValidityHours = *override.TemporaryPasswordValidityHours
	}
	return policy
}

// commonPasswordBlocklist contains a curated embedded common-password list.
//
// Matching is EXACT against a normalized form (see normalizeForBlocklist), never
// substring — substring matching would reject any password that merely contains
// a common word, which is both user-hostile and no more secure. Normalization is
// what gives a list this size real reach: undoing leet substitutions and the
// trailing "1!" collapses the whole "Password1!", "P@ssw0rd", "Passw0rd!" family
// onto the single entry "password".
//
// The list is still well short of the ~10^4 entries OWASP ASVS V2.1.7 expects.
// To close that: drop the SecLists "10-million-password-list-top-10000.txt" into
// common_passwords.txt (one entry per line, lowercase). It is a data swap — no
// code change — because entries are loaded from the file at init below.
//
//go:embed common_passwords.txt
var commonPasswordsFile string

var commonPasswordBlocklist = func() map[string]struct{} {
	m := map[string]struct{}{
		"123456":        {},
		"123456789":     {},
		"12345":         {},
		"qwerty":        {},
		"password":      {},
		"12345678":      {},
		"111111":        {},
		"123123":        {},
		"1234567890":    {},
		"1234567":       {},
		"qwerty123":     {},
		"000000":        {},
		"1q2w3e":        {},
		"aa12345678":    {},
		"abc123":        {},
		"password1":     {},
		"1234":          {},
		"qwertyuiop":    {},
		"123321":        {},
		"password123":   {},
		"1q2w3e4r5t":    {},
		"iloveyou":      {},
		"654321":        {},
		"666666":        {},
		"987654321":     {},
		"123":           {},
		"monkey":        {},
		"dragon":        {},
		"letmein":       {},
		"football":      {},
		"baseball":      {},
		"sunshine":      {},
		"princess":      {},
		"admin":         {},
		"welcome":       {},
		"login":         {},
		"solo":          {},
		"starwars":      {},
		"master":        {},
		"hello":         {},
		"freedom":       {},
		"whatever":      {},
		"qazwsx":        {},
		"trustno1":      {},
		"jordan":        {},
		"harley":        {},
		"buster":        {},
		"thomas":        {},
		"tigger":        {},
		"robert":        {},
		"soccer":        {},
		"hockey":        {},
		"killer":        {},
		"george":        {},
		"charlie":       {},
		"andrew":        {},
		"michelle":      {},
		"love":          {},
		"jessica":       {},
		"pepper":        {},
		"daniel":        {},
		"access":        {},
		"shadow":        {},
		"maggie":        {},
		"computer":      {},
		"ashley":        {},
		"bailey":        {},
		"passw0rd":      {},
		"superman":      {},
		"michael":       {},
		"football1":     {},
		"q1w2e3r4":      {},
		"zaq12wsx":      {},
		"password!":     {},
		"password1!":    {},
		"password123!":  {},
		"password1234!": {},
		"welcome1":      {},
		"welcome1!":     {},
		"admin123":      {},
		"admin123!":     {},
		"changeme":      {},
		"changeme1":     {},
		"changeme1!":    {},
		"letmein1":      {},
		"letmein1!":     {},
		"default":       {},
		"default1":      {},
		"default1!":     {},
		"maintainerd":   {},
		"maintainerd1":  {},
		"maintainerd1!": {},
		"company123":    {},
		"company123!":   {},
		"summer2024!":   {},
		"summer2025!":   {},
		"summer2026!":   {},
		"winter2024!":   {},
		"winter2025!":   {},
		"winter2026!":   {},
		"spring2024!":   {},
		"spring2025!":   {},
		"spring2026!":   {},
		"autumn2024!":   {},
		"autumn2025!":   {},
		"autumn2026!":   {},
		"fall2024!":     {},
		"fall2025!":     {},
		"fall2026!":     {},
	}
	for _, line := range strings.Split(commonPasswordsFile, "\n") {
		if p := strings.TrimSpace(strings.ToLower(line)); p != "" {
			m[p] = struct{}{}
		}
	}
	return m
}()

// leetSubstitutions maps a visual substitution back to the letters it can stand
// in for, so a disguised blocklist entry still matches. Without this the
// blocklist only ever caught the literal spelling: "password" was rejected while
// "P@ssw0rd" — the single most common corporate password shape there is — sailed
// straight through.
//
// Note that '1' is genuinely ambiguous ("l" in "l33t", "i" in "adm1n"), so it
// expands to both. That is why this cannot be a single deterministic rewrite:
// collapsing it to one choice silently loses half the matches.
var leetSubstitutions = map[rune][]rune{
	'@': {'a'},
	'4': {'a'},
	'8': {'b'},
	'(': {'c'},
	'3': {'e'},
	'6': {'g'},
	'1': {'l', 'i'},
	'!': {'i'},
	'0': {'o'},
	'5': {'s'},
	'$': {'s'},
	'7': {'t'},
	'+': {'t'},
	'2': {'z'},
}

// maxLeetVariants caps the substitution fan-out. Each ambiguous character
// doubles the candidate count, so an unbounded expansion would turn a long
// digit-heavy password into a denial of service.
const maxLeetVariants = 32

// blocklistCandidates returns every normalized form of a password worth testing
// against the blocklist.
//
// It deliberately keeps the UNSUBSTITUTED form alongside the substituted ones.
// Applying leet rewriting unconditionally is wrong in the other direction: it
// would turn the real blocklist entry "summer2026!" into "summerzozgi" and stop
// matching it. Both readings have to be tried.
//
// Matching against these candidates stays EXACT, never substring — substring
// matching would reject any password that merely contains a common word, which
// is user-hostile and buys nothing.
func blocklistCandidates(password string) []string {
	base := strings.ToLower(strings.TrimSpace(password))
	if base == "" {
		return nil
	}

	// The trailing run of digits and punctuation that composition rules push
	// people into appending. Bounded at 5 so a long numeric password is not
	// whittled down to an unrelated stem, but wide enough to cover the two
	// dominant shapes: "Password1!" and "Password!2026" (symbol plus year).
	stems := []string{base}
	if stem := strings.TrimRight(base, "0123456789!@#$%^&*()_+-=.?"); stem != base &&
		stem != "" && len(base)-len(stem) <= 5 {
		stems = append(stems, stem)
	}

	seen := make(map[string]struct{}, len(stems)*2)
	var candidates []string
	for _, stem := range stems {
		for _, variant := range leetVariants(stem) {
			if _, dup := seen[variant]; dup {
				continue
			}
			seen[variant] = struct{}{}
			candidates = append(candidates, variant)
		}
	}
	return candidates
}

// leetVariants expands every combination of substitutions for s, including s
// itself, up to maxLeetVariants.
func leetVariants(s string) []string {
	variants := []string{""}
	for _, r := range s {
		replacements, substitutable := leetSubstitutions[r]
		next := make([]string, 0, len(variants)*(len(replacements)+1))
		for _, prefix := range variants {
			next = append(next, prefix+string(r))
			if !substitutable {
				continue
			}
			for _, replacement := range replacements {
				next = append(next, prefix+string(replacement))
			}
		}
		// Truncate rather than merely stopping the fan-out: without this the
		// set still grows by one entry per prefix per character, so a long
		// digit-heavy password walks past the cap anyway.
		if len(next) > maxLeetVariants {
			next = next[:maxLeetVariants]
		}
		variants = next
	}
	return variants
}

// isBlocklisted reports whether the password matches a blocklist entry under any
// normalization, and whether reaching that match required undoing a disguise.
// The second return value is what the strength estimator charges extra for.
func isBlocklisted(password string) (matched bool, disguised bool) {
	literal := strings.ToLower(strings.TrimSpace(password))
	for _, candidate := range blocklistCandidates(password) {
		if _, found := commonPasswordBlocklist[candidate]; found {
			return true, candidate != literal
		}
	}
	return false, false
}

// PasswordUserContext carries the identity values a password must not simply
// restate. OWASP ASVS V2.1.4 and NIST 800-63B both require context-specific word
// rejection; without it "acme-corp" is a perfectly policy-compliant password at
// tenant Acme Corp. All fields are optional — empty ones are skipped.
type PasswordUserContext struct {
	Username   string
	Email      string
	FirstName  string
	LastName   string
	TenantName string
}

// contextTerm pairs a normalized identity value with the label shown to the user
// when it matches. The label never echoes the value itself.
type contextTerm struct {
	label string
	value string
}

// contextTerms returns the normalized identity terms worth checking. The local
// part of the email is included separately from the whole address because that
// is the part people actually reuse as a password.
func (c PasswordUserContext) contextTerms() []contextTerm {
	raw := []contextTerm{
		{"username", c.Username},
		{"email address", c.Email},
		{"first name", c.FirstName},
		{"last name", c.LastName},
		{"organization name", c.TenantName},
	}
	if local, _, found := strings.Cut(c.Email, "@"); found {
		raw = append(raw, contextTerm{"email address", local})
	}

	terms := make([]contextTerm, 0, len(raw))
	for _, term := range raw {
		value := strings.ToLower(strings.TrimSpace(term.value))
		// Terms shorter than 4 characters produce false positives on ordinary
		// passwords — a user named "Al" would veto anything containing "al".
		if len(value) < 4 {
			continue
		}
		terms = append(terms, contextTerm{label: term.label, value: value})
	}
	return terms
}

// matchesContextTerm reports the label of the first identity term the password
// contains, or "" when none do. Both the literal password and its de-disguised
// form are checked, so "Acme-Corp-2026" is caught at tenant "Acme Corp" the same
// way "acmecorp" is.
func matchesContextTerm(password string, userCtx PasswordUserContext) string {
	terms := userCtx.contextTerms()
	if len(terms) == 0 {
		return ""
	}
	haystacks := append([]string{strings.ToLower(password)}, blocklistCandidates(password)...)
	for _, term := range terms {
		// Compare against the term with separators removed too: "acme corp"
		// and "acme-corp" are the same term as far as a password is concerned.
		needles := []string{term.value, stripSeparators(term.value)}
		for _, needle := range needles {
			if len(needle) < 4 {
				continue
			}
			for _, haystack := range haystacks {
				if strings.Contains(haystack, needle) || strings.Contains(stripSeparators(haystack), needle) {
					return term.label
				}
			}
		}
	}
	return ""
}

func stripSeparators(s string) string {
	return strings.NewReplacer(" ", "", "-", "", "_", "", ".", "").Replace(s)
}

// ValidatePasswordPolicy validates a password against the given policy.
func ValidatePasswordPolicy(password string, policy PasswordPolicy) error {
	return ValidatePasswordPolicyWithContext(context.Background(), password, policy)
}

// ValidatePasswordPolicyWithContext validates a password against the given policy,
// including HIBP k-anonymity checks which require a context for tracing.
func ValidatePasswordPolicyWithContext(ctx context.Context, password string, policy PasswordPolicy) error {
	return ValidatePasswordPolicyForUser(ctx, password, policy, PasswordUserContext{})
}

// ValidatePasswordPolicyForUser validates a password against the given policy and
// additionally rejects passwords that restate the user's own identity. Callers
// that know who the password belongs to should prefer this over
// ValidatePasswordPolicyWithContext.
func ValidatePasswordPolicyForUser(ctx context.Context, password string, policy PasswordPolicy, userCtx PasswordUserContext) error {
	passwordLength := utf8.RuneCountInString(password)
	if passwordLength < policy.MinLength {
		return fmt.Errorf("password must be at least %d characters long", policy.MinLength)
	}
	if policy.MaxLength > 0 && passwordLength > policy.MaxLength {
		return fmt.Errorf("password must not exceed %d characters", policy.MaxLength)
	}
	if policy.RequireUpper && !reUpper.MatchString(password) {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if policy.RequireLower && !reLower.MatchString(password) {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if policy.RequireDigit && !reDigit.MatchString(password) {
		return fmt.Errorf("password must contain at least one digit")
	}
	if policy.RequireSpecial && !reSpecial.MatchString(password) {
		return fmt.Errorf("password must contain at least one special character")
	}
	if policy.BlocklistEnabled {
		// Matches the literal spelling and every de-disguised reading, so
		// "password", "P@ssw0rd" and "Password1!" all hit the same list entry.
		if blocked, _ := isBlocklisted(password); blocked {
			return fmt.Errorf("password is a common weak password")
		}
	}
	// Context-specific words are checked whenever the caller supplies identity,
	// independent of BlocklistEnabled: a password that is just the username is
	// not a "common password" problem, it is an account-takeover problem.
	if term := matchesContextTerm(password, userCtx); term != "" {
		return fmt.Errorf("password must not contain your %s", term)
	}
	if policy.MinStrengthScore > 0 {
		score := PasswordStrengthScore(password)
		if score < policy.MinStrengthScore {
			return fmt.Errorf("password strength score %d is below the required minimum of %d", score, policy.MinStrengthScore)
		}
	}
	if policy.CheckHIBP {
		if CheckHIBPPassword(ctx, []byte(password)) {
			return fmt.Errorf("password was found in known data breaches")
		}
	}
	return nil
}
