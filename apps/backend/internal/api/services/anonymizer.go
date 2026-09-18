package services

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Redaction tokens replace PII before the text is sent to the external LLM
// (A8). They are neutral, so the model can still analyze the sentence grammar.
const (
	emailToken = "[email]"
	phoneToken = "[phone]"
	nameToken  = "[name]"
)

var (
	emailPattern = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)

	// phonePattern matches international, parenthesized and separator-grouped
	// numbers. A bare four-digit year (e.g. 2024) is intentionally not matched:
	// the grouped alternative requires a 2-3 digit leading group plus a
	// separator, so ISO dates such as 2026-09-18 are left untouched.
	phonePattern = regexp.MustCompile(
		`(?:\+\d{1,3}(?:[\s.\-]?\(?\d{1,4}\)?){2,})` +
			`|(?:\(\d{1,4}\)[\s.\-]?\d{2,4}(?:[\s.\-]?\d{2,4})?\b)` +
			`|(?:\b\d{2,3}[\s.\-]\d{2,4}(?:[\s.\-]?\d{2,4})?\b)`,
	)

	// wordPattern yields Unicode letter runs so names can be matched as whole
	// words without splitting on accents.
	wordPattern = regexp.MustCompile(`[\p{L}]+`)

	// capitalizedPattern matches a single capitalized word (accent aware). Used
	// only for the Spanish source, where a mid-sentence capitalized word is
	// most likely a proper noun.
	capitalizedPattern = regexp.MustCompile(`\b[A-ZÁÉÍÓÚÜÑ][a-záéíóúüñ]+\b`)
)

// commonNames is a curated seed list of common Spanish and English given
// names. It is the deterministic half of name detection (A8); the Spanish
// capitalization heuristic below covers unknown names.
var commonNames = func() map[string]struct{} {
	names := []string{
		// Spanish
		"maría", "maria", "jose", "josé", "juan", "ana", "luis", "carlos", "laura", "carmen",
		"javier", "miguel", "sofía", "sofia", "lucía", "lucia", "pedro", "pablo",
		"marta", "elena", "david", "daniel", "sara", "paula", "andrés", "andres",
		"diego", "isabel", "rosa", "antonio", "manuel", "francisco", "teresa",
		"beatriz", "cristina", "jorge", "raúl", "raul", "álvaro", "alvaro", "adrián",
		"adrian", "irene", "natalia", "silvia", "patricia", "fernando", "alberto",
		"ricardo", "sergio", "víctor", "victor", "raquel", "julia", "nuria", "clara",
		"marcos", "hugo", "mateo", "martín", "martin", "lucas", "alejandro", "gabriel",
		// English
		"john", "mary", "james", "robert", "jennifer", "michael", "linda", "william",
		"elizabeth", "barbara", "richard", "susan", "joseph", "jessica", "thomas",
		"sarah", "charles", "karen", "christopher", "nancy", "lisa", "matthew",
		"betty", "anthony", "margaret", "mark", "sandra", "donald", "ashley",
		"steven", "kimberly", "paul", "emily", "andrew", "donna", "joshua",
		"michelle", "kenneth", "carol", "kevin", "amanda", "brian", "dorothy",
		"george", "melissa", "edward", "deborah", "ronald", "stephanie", "timothy",
		"rebecca", "jason", "sharon", "jeffrey", "cynthia", "jacob", "kathleen",
		"gary", "amy", "nicholas", "angela", "eric", "anna", "stephen", "brenda",
		"pamela", "justin", "emma", "scott", "nicole", "brandon", "helen",
		"benjamin", "samantha", "samuel", "katherine", "gregory", "christine",
		"alexander", "debra", "patrick", "rachel", "frank", "carolyn", "raymond",
		"janet", "jack", "catherine", "dennis", "jerry", "olivia", "tyler",
		"heather", "aaron", "diane", "adam", "julie", "nathan", "joyce", "henry",
		"victoria", "zachary", "ruth", "douglas", "virginia", "peter", "lauren",
		"kyle", "christina", "noah", "joan", "ethan", "evelyn", "jeremy", "judith",
		"walter", "megan", "christian", "cheryl", "keith", "andrea", "roger",
		"hannah", "terry", "martha", "austin", "gloria", "sean", "gerald", "ann",
		"carl", "harold", "diana", "marie", "alice", "bruce", "roy", "ralph",
		"eugene", "randy", "louis", "bobby", "howard", "eugene",
	}
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		set[name] = struct{}{}
	}
	return set
}()

// Anonymize removes PII (emails, phone numbers and known proper names) from
// text before it is sent to the external LLM (A8). It is the general path and
// is safe for the English draft, where the capitalization heuristic is not
// applied to avoid redacting ordinary capitalized words.
func Anonymize(text string) string {
	return redactNames(redactPhones(redactEmails(text)))
}

// AnonymizeSpanish applies Anonymize plus the Spanish capitalization
// heuristic: a capitalized word that is not sentence-initial is treated as a
// proper noun and redacted. Use it for the Spanish source text (es).
func AnonymizeSpanish(text string) string {
	return redactCapitalized(Anonymize(text))
}

// redactEmails replaces every email address with the email token.
func redactEmails(text string) string {
	return emailPattern.ReplaceAllString(text, emailToken)
}

// redactPhones replaces every phone-like number with the phone token.
func redactPhones(text string) string {
	return phonePattern.ReplaceAllString(text, phoneToken)
}

// redactNames replaces capitalized whole words that belong to the curated name
// list, accent-aware (strings.ToLower is Unicode aware). Requiring the leading
// uppercase distinguishes proper names from common words that collide with them
// (e.g. "mark" the verb vs "Mark" the name).
func redactNames(text string) string {
	return wordPattern.ReplaceAllStringFunc(text, func(word string) string {
		if !startsWithUpper(word) {
			return word
		}
		if _, ok := commonNames[strings.ToLower(word)]; ok {
			return nameToken
		}
		return word
	})
}

// startsWithUpper reports whether the word begins with an uppercase letter.
func startsWithUpper(word string) bool {
	r, _ := utf8.DecodeRuneInString(word)
	return unicode.IsUpper(r)
}

// redactCapitalized replaces capitalized words that appear mid-sentence (not
// at the start of the text and not just after a sentence boundary) with the
// name token.
func redactCapitalized(text string) string {
	matches := capitalizedPattern.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return text
	}

	var b strings.Builder
	last := 0
	for _, match := range matches {
		start, end := match[0], match[1]
		if isSentenceInitial(text, start) {
			continue
		}
		b.WriteString(text[last:start])
		b.WriteString(nameToken)
		last = end
	}
	b.WriteString(text[last:])
	return b.String()
}

// isSentenceInitial reports whether position start begins a sentence: the
// text start or a letter preceded only by whitespace after a sentence boundary.
func isSentenceInitial(text string, start int) bool {
	i := start - 1
	for i >= 0 && (text[i] == ' ' || text[i] == '\t') {
		i--
	}
	if i < 0 {
		return true
	}
	switch text[i] {
	case '.', '!', '?', ':', ';', '\n', '\r':
		return true
	default:
		return false
	}
}
