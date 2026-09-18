package services

import "testing"

func TestAnonymize_Email_IsRedacted(t *testing.T) {
	got := Anonymize("Escribe a maria.lopez@example.com hoy.")
	want := "Escribe a [email] hoy."
	if got != want {
		t.Fatalf("Anonymize() = %q, want %q", got, want)
	}
}

func TestAnonymize_Phone_IsRedacted(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"international", "Llámame al +34 600 123 456.", "Llámame al [phone]."},
		{"us international", "Call +1 202 555 0100 now.", "Call [phone] now."},
		{"parenthesized", "Call (555) 123-4567 now.", "Call [phone] now."},
		{"dashed local", "Call 555-1234 now.", "Call [phone] now."},
		{"spaced local", "Llama al 600 123 456.", "Llama al [phone]."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Anonymize(tc.in); got != tc.want {
				t.Fatalf("Anonymize(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestAnonymize_Year_IsNotRedacted(t *testing.T) {
	if got, want := Anonymize("En 2024 viví allí."), "En 2024 viví allí."; got != want {
		t.Fatalf("Anonymize() = %q, want %q", got, want)
	}
}

func TestAnonymize_KnownName_IsRedactedWholeWord(t *testing.T) {
	if got, want := Anonymize("Mi amiga Carmen vive aquí."), "Mi amiga [name] vive aquí."; got != want {
		t.Fatalf("Anonymize() = %q, want %q", got, want)
	}
}

func TestAnonymize_CapitalizedName_IsRedacted(t *testing.T) {
	if got, want := Anonymize("Mark wrote a letter."), "[name] wrote a letter."; got != want {
		t.Fatalf("Anonymize() = %q, want %q", got, want)
	}
}

func TestAnonymize_LowercaseCommonWordsMatchingNames_AreKept(t *testing.T) {
	for _, in := range []string{
		"Please mark the correct answer.",
		"Be frank with me.",
		"They sing a carol.",
	} {
		if got := Anonymize(in); got != in {
			t.Fatalf("Anonymize(%q) = %q, want unchanged", in, got)
		}
	}
}

func TestAnonymize_UnknownNameInList_IsNotRedactedWithoutSpanishHeuristic(t *testing.T) {
	// Barcelona is not in the curated list; the generic path must not use the
	// capitalization heuristic (that would corrupt the English draft).
	if got, want := Anonymize("Yesterday I visited Barcelona."), "Yesterday I visited Barcelona."; got != want {
		t.Fatalf("Anonymize() = %q, want %q", got, want)
	}
}

func TestAnonymize_NoPII_IsUnchanged(t *testing.T) {
	in := "El perro corre por el parque."
	if got := Anonymize(in); got != in {
		t.Fatalf("Anonymize() = %q, want %q", got, in)
	}
}

func TestAnonymize_Empty_IsUnchanged(t *testing.T) {
	if got := Anonymize(""); got != "" {
		t.Fatalf("Anonymize() = %q, want empty", got)
	}
}

func TestAnonymize_MixedPII_AllRedacted(t *testing.T) {
	got := Anonymize("Carmen escribe a carmen@example.com o llama al 600 123 456.")
	want := "[name] escribe a [email] o llama al [phone]."
	if got != want {
		t.Fatalf("Anonymize() = %q, want %q", got, want)
	}
}

func TestAnonymizeSpanish_MidSentenceCapitalized_IsTreatedAsName(t *testing.T) {
	got := AnonymizeSpanish("Ayer visité Barcelona con mis primos.")
	want := "Ayer visité [name] con mis primos."
	if got != want {
		t.Fatalf("AnonymizeSpanish() = %q, want %q", got, want)
	}
}

func TestAnonymizeSpanish_SentenceInitialCapitalized_IsKept(t *testing.T) {
	// A capitalized word at the start of a sentence is capitalization by rule,
	// not necessarily a proper name.
	got := AnonymizeSpanish("Barcelona es una ciudad grande.")
	want := "Barcelona es una ciudad grande."
	if got != want {
		t.Fatalf("AnonymizeSpanish() = %q, want %q", got, want)
	}
}

func TestAnonymizeSpanish_CapitalizedAfterPeriod_IsKept(t *testing.T) {
	got := AnonymizeSpanish("Fui allí. Barcelona me encantó.")
	want := "Fui allí. Barcelona me encantó."
	if got != want {
		t.Fatalf("AnonymizeSpanish() = %q, want %q", got, want)
	}
}

func TestAnonymizeSpanish_KnownNameInList_IsRedactedRegardlessOfPosition(t *testing.T) {
	got := AnonymizeSpanish("Carmen llegó tarde.")
	want := "[name] llegó tarde."
	if got != want {
		t.Fatalf("AnonymizeSpanish() = %q, want %q", got, want)
	}
}

func TestAnonymizeSpanish_LowercaseWord_IsNotRedacted(t *testing.T) {
	in := "Hablo español con mis amigos."
	if got := AnonymizeSpanish(in); got != in {
		t.Fatalf("AnonymizeSpanish() = %q, want %q", got, in)
	}
}
