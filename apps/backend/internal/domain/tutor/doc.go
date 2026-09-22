// Package tutor is the fifth bounded context of LangLint: the adaptive tutor
// (PRODUCT_DOMAIN §12.1, checklist 08). It generates personalized study sessions
// from the learner's historical error patterns (spaced repetition).
//
// Isolation (A3): like the other bounded contexts, tutor only imports the Go
// standard library and the shared root domain package (ID, ErrorPattern,
// errors). It never imports practice, analysis or analytics, and it never holds
// raw practice or analysis data: it consumes aggregated weakness data (8.1.4).
package tutor
