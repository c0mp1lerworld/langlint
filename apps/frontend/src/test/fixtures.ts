import type { components } from "@/lib/api/gen";

type Schemas = components["schemas"];

export function targetRule(over: Partial<Schemas["TargetRule"]> = {}): Schemas["TargetRule"] {
  return { verb: "run", ...over };
}

export function practice(over: Partial<Schemas["Practice"]> = {}): Schemas["Practice"] {
  return {
    id: "11111111-1111-7111-8111-111111111111",
    user_id: "22222222-2222-7222-8222-222222222222",
    source_text: "El perro escapó.",
    draft_text: "The dog escaped.",
    target_rules: [targetRule()],
    status: "completed",
    created_at: "2026-09-19T00:00:00Z",
    updated_at: "2026-09-19T00:00:00Z",
    ...over,
  };
}

export function fragment(over: Partial<Schemas["Fragment"]> = {}): Schemas["Fragment"] {
  return {
    source_es: "Ayer fui al parque.",
    user_draft: "Yesterday I go to the park.",
    correction: "Yesterday I went to the park.",
    target_verb_review: {
      verb: "go",
      correct_form: "went",
      rule: "pasado simple de un verbo irregular",
      why: "la acción ocurrió ayer",
      es_contrast: "en español el pretérito cambia la forma",
      alternatives: ["went", "did go (énfasis)"],
    },
    lexical_clarification: {
      term: "go",
      meaning: "ir",
      why_wrong: "el borrador usa el presente",
      alternatives: [],
    },
    grammar_explanation: {
      rule_name: "pasado simple irregular",
      explanation: "el verbo no añade -ed, sino que cambia de forma",
      construction: "go → went",
      counterexample: "I go → I went",
      exception: "los regulares añaden -ed",
      es_contrast: "el español usa 'fui'",
    },
    error_patterns: [{ code: "tense_agreement", severity: "critical", note: "pasado" }],
    ...over,
  };
}

export function analysis(over: Partial<Schemas["Analysis"]> = {}): Schemas["Analysis"] {
  return {
    id: "33333333-3333-7333-8333-333333333333",
    practice_id: "11111111-1111-7111-8111-111111111111",
    model: "gpt-4o-mini",
    model_version: "gpt-4o-mini-2024-07-18",
    status: "completed",
    fragments: [fragment()],
    ...over,
  };
}

export function errorPatternAggregate(
  over: Partial<Schemas["ErrorPatternAggregate"]> = {},
): Schemas["ErrorPatternAggregate"] {
  return { code: "tense_agreement", count: 2, last_seen_at: "2026-09-19T00:00:00Z", ...over };
}
