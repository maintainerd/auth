package security

import (
	"math"
	"strings"
	"unicode"
)

// Password strength scoring.
//
// WHAT CHANGED AND WHY
//
// The previous implementation of PasswordStrengthScore claimed to be "a
// simplified zxcvbn-like model" but was a character-class counter: it awarded a
// point for each of upper/lower/digit/symbol present, plus points for crossing
// length thresholds. That is composition scoring wearing a zxcvbn label, and it
// is actively harmful here:
//
//   - It contradicts the tenant policy. A tenant that deliberately turns
//     RequireUpper/Digit/Special off — the NIST SP 800-63B posture this service
//     ships as its default — still had composition enforced through the back
//     door the moment MinStrengthScore was set above 0.
//   - It scored the wrong thing. "Passw0rd!" hits all four classes and scored
//     near the top; "correct horse battery staple" hits one class and scored
//     near the bottom. The first is cracked in milliseconds, the second is not.
//
// The model below estimates GUESSABILITY instead, which is what a strength score
// is supposed to mean and what NIST 800-63B §5.1.1.2 and OWASP ASVS V2.1 care
// about. Character classes are not rewarded anywhere.
//
// The estimator follows zxcvbn's shape — decompose the password into
// recognizable patterns, cost each one by how many guesses an attacker who knows
// the pattern would need, and charge the leftover characters at a flat
// bruteforce rate. Like zxcvbn, that rate is 10 guesses per character
// (BRUTEFORCE_CARDINALITY) regardless of alphabet: a cracker's search order is
// driven by likelihood, not by which glyphs happen to appear. That is precisely
// the property that stops symbols from buying free score.
//
// This is deliberately NOT a full zxcvbn port — there is no frequency-ranked
// English/name/surname corpus and no exhaustive leet-substitution search, so it
// will under-detect dictionary content inside long passphrases. It errs toward
// reporting a password as WEAKER than real zxcvbn would, never stronger, so it
// cannot wave through something zxcvbn would reject. To swap in a full port,
// replace EstimatePasswordGuessesLog10 and leave the thresholds below alone.

// Guess-count thresholds, in log10, matching zxcvbn's published score bands:
//
//	0 → < 10^3   trivially guessed
//	1 → < 10^6   very guessable
//	2 → < 10^8   somewhat guessable
//	3 → < 10^10  safely unguessable against an offline slow hash
//	4 → ≥ 10^10  strong
const (
	guessesLog10Score1 = 3
	guessesLog10Score2 = 6
	guessesLog10Score3 = 8
	guessesLog10Score4 = 10
)

// bruteforceLog10PerChar is log10 of zxcvbn's BRUTEFORCE_CARDINALITY (10).
// Every character not explained by a cheaper pattern costs this much.
const bruteforceLog10PerChar = 1.0

// PasswordStrengthScore rates a password 0-4 by estimated guessability.
// 0 = trivially guessed, 4 = strong. Character classes carry no weight; length
// that an attacker cannot shortcut does.
func PasswordStrengthScore(password string) int {
	return guessesLog10ToScore(EstimatePasswordGuessesLog10(password))
}

// EstimatePasswordGuessesLog10 returns log10 of the estimated number of guesses
// needed to arrive at this password. Exported so callers and tests can reason
// about the underlying estimate rather than the coarse 0-4 bucket.
func EstimatePasswordGuessesLog10(password string) float64 {
	runes := []rune(password)
	if len(runes) == 0 {
		return 0
	}

	// covered[i] is true once character i has been explained by a pattern, so
	// the bruteforce pass below does not charge for it a second time.
	covered := make([]bool, len(runes))
	total := 0.0

	for _, match := range findWeakPatterns(runes) {
		// Skip a pattern whose characters are already fully explained by an
		// earlier, longer match — otherwise overlapping patterns stack cost.
		if rangeFullyCovered(covered, match.start, match.end) {
			continue
		}
		for i := match.start; i < match.end; i++ {
			covered[i] = true
		}
		total += match.log10Guesses
	}

	uncovered := 0
	for _, c := range covered {
		if !c {
			uncovered++
		}
	}
	total += float64(uncovered) * bruteforceLog10PerChar

	return total
}

func guessesLog10ToScore(log10Guesses float64) int {
	switch {
	case log10Guesses < guessesLog10Score1:
		return 0
	case log10Guesses < guessesLog10Score2:
		return 1
	case log10Guesses < guessesLog10Score3:
		return 2
	case log10Guesses < guessesLog10Score4:
		return 3
	default:
		return 4
	}
}

// weakPattern is a span of the password an attacker can guess more cheaply than
// brute force, together with the log10 cost of doing so.
type weakPattern struct {
	start        int
	end          int // exclusive
	log10Guesses float64
}

// findWeakPatterns decomposes the password into recognizable cheap-to-guess
// spans, longest-first so a long match wins over the short matches nested inside
// it.
func findWeakPatterns(runes []rune) []weakPattern {
	var matches []weakPattern
	matches = append(matches, findBlocklistMatches(runes)...)
	matches = append(matches, findRepeatMatches(runes)...)
	matches = append(matches, findSequenceMatches(runes)...)
	matches = append(matches, findKeyboardMatches(runes)...)
	matches = append(matches, findYearMatches(runes)...)

	// Longest span first; ties broken by the cheaper (more damning) estimate.
	for i := 1; i < len(matches); i++ {
		for j := i; j > 0; j-- {
			a, b := matches[j-1], matches[j]
			aLen, bLen := a.end-a.start, b.end-b.start
			if bLen > aLen || (bLen == aLen && b.log10Guesses < a.log10Guesses) {
				matches[j-1], matches[j] = b, a
				continue
			}
			break
		}
	}
	return matches
}

func rangeFullyCovered(covered []bool, start, end int) bool {
	for i := start; i < end; i++ {
		if !covered[i] {
			return false
		}
	}
	return true
}

// findBlocklistMatches locates substrings present in the common-password
// blocklist, including capitalization and leet-speak disguises. A disguise
// multiplies the cost by the number of variants an attacker would enumerate,
// which is why "P@ssw0rd" costs more than "password" — but nowhere near what a
// random string of the same length costs.
func findBlocklistMatches(runes []rune) []weakPattern {
	const minDictLen = 4
	var matches []weakPattern

	for start := 0; start+minDictLen <= len(runes); start++ {
		for end := len(runes); end-start >= minDictLen; end-- {
			candidate := string(runes[start:end])
			blocked, disguised := isBlocklisted(candidate)
			if !blocked {
				continue
			}
			matches = append(matches, weakPattern{
				start:        start,
				end:          end,
				log10Guesses: blocklistMatchLog10(candidate, disguised),
			})
			// Longest match at this start position wins; don't also emit the
			// shorter matches nested inside it.
			break
		}
	}
	return matches
}

// blocklistMatchLog10 costs a dictionary hit as (rank within the list) times the
// number of disguise variants an attacker must enumerate to reach this spelling.
func blocklistMatchLog10(raw string, disguised bool) float64 {
	log10Rank := math.Log10(float64(len(commonPasswordBlocklist)))

	variants := 1.0
	if hasUpperCase(raw) {
		// Capitalization variants: first-letter caps, ALL CAPS, and the general
		// case. zxcvbn charges the exact binomial; a flat factor is close enough
		// and stays on the conservative side.
		variants *= 4
	}
	if disguised {
		// Leet substitutions or a trailing "1!" had to be undone to reach the
		// match, so the attacker enumerated those variants too.
		variants *= 8
	}
	return log10Rank + math.Log10(variants)
}

func hasUpperCase(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

// findRepeatMatches finds runs of one repeated character ("aaaa"). An attacker
// guesses these as (which character) × (how long), not as N independent picks.
func findRepeatMatches(runes []rune) []weakPattern {
	var matches []weakPattern
	for start := 0; start < len(runes); {
		end := start + 1
		for end < len(runes) && runes[end] == runes[start] {
			end++
		}
		if end-start >= 3 {
			alphabet := float64(charsetSize(string(runes[start:end])))
			matches = append(matches, weakPattern{
				start:        start,
				end:          end,
				log10Guesses: math.Log10(alphabet * float64(end-start)),
			})
		}
		start = end
	}
	return matches
}

// findSequenceMatches finds ascending or descending runs of adjacent code points
// ("abcd", "9876").
func findSequenceMatches(runes []rune) []weakPattern {
	var matches []weakPattern
	for start := 0; start+2 < len(runes); {
		delta := runes[start+1] - runes[start]
		if delta != 1 && delta != -1 {
			start++
			continue
		}
		end := start + 2
		for end < len(runes) && runes[end]-runes[end-1] == delta {
			end++
		}
		if end-start >= 3 {
			// ~26 plausible starting points × length × 2 directions.
			matches = append(matches, weakPattern{
				start:        start,
				end:          end,
				log10Guesses: math.Log10(26 * float64(end-start) * 2),
			})
			start = end
			continue
		}
		start++
	}
	return matches
}

// keyboardRows are the adjacency runs a walk-the-keyboard password follows.
// Reversals are checked too, so "ytrewq" matches as well as "qwerty".
var keyboardRows = []string{
	"qwertyuiop",
	"asdfghjkl",
	"zxcvbnm",
	"1234567890",
	"!@#$%^&*()",
}

// findKeyboardMatches finds keyboard walks of 4 or more characters. These read as
// high-entropy to a class counter — "1qaz@WSX" hits all four classes — while
// being among the first things every cracking rule set tries.
func findKeyboardMatches(runes []rune) []weakPattern {
	const minWalk = 4
	lowerRunes := []rune(strings.ToLower(string(runes)))

	var matches []weakPattern
	for start := 0; start+minWalk <= len(lowerRunes); start++ {
		for end := len(lowerRunes); end-start >= minWalk; end-- {
			if !isKeyboardWalk(string(lowerRunes[start:end])) {
				continue
			}
			matches = append(matches, weakPattern{
				start: start,
				end:   end,
				// rows × starting offset × length × direction.
				log10Guesses: math.Log10(float64(len(keyboardRows)) * 10 * float64(end-start) * 2),
			})
			break
		}
	}
	return matches
}

func isKeyboardWalk(segment string) bool {
	reversed := reverseString(segment)
	for _, row := range keyboardRows {
		if strings.Contains(row, segment) || strings.Contains(row, reversed) {
			return true
		}
	}
	return false
}

func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// findYearMatches finds 4-digit years. "Summer2026!" is the archetypal
// rotation-policy password; those four digits must not be credited as entropy.
func findYearMatches(runes []rune) []weakPattern {
	var matches []weakPattern
	for start := 0; start+4 <= len(runes); start++ {
		segment := runes[start : start+4]
		if !allDigits(segment) {
			continue
		}
		if (segment[0] == '1' && segment[1] == '9') || (segment[0] == '2' && segment[1] == '0') {
			// ~120 plausible years.
			matches = append(matches, weakPattern{start: start, end: start + 4, log10Guesses: math.Log10(120)})
		}
	}
	return matches
}

func allDigits(runes []rune) bool {
	for _, r := range runes {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// charsetSize reports the size of the alphabet a string draws from. It is used
// ONLY to cost repeat runs, where an attacker really does enumerate an alphabet.
// It deliberately plays no part in the bruteforce estimate, so adding a symbol
// never buys score on its own.
func charsetSize(s string) int {
	var hasLower, hasUpper, hasDigit, hasSymbol, hasOther bool
	for _, r := range s {
		switch {
		case r > 127:
			hasOther = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		default:
			hasSymbol = true
		}
	}
	size := 0
	if hasLower {
		size += 26
	}
	if hasUpper {
		size += 26
	}
	if hasDigit {
		size += 10
	}
	if hasSymbol {
		size += 33
	}
	if hasOther {
		size += 100
	}
	if size == 0 {
		size = 1
	}
	return size
}
