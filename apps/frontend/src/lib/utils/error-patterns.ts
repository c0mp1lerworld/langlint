import type { components } from "@/lib/api/gen";

export type ErrorPatternCode = components["schemas"]["ErrorPatternCode"];
export type ErrorPatternAggregate = components["schemas"]["ErrorPatternAggregate"];

export const ERROR_PATTERN_LABELS: Record<ErrorPatternCode, string> = {
  infinitive_conjugation: "infinitivo/conjugación",
  passive_voice_misuse: "voz pasiva",
  idiom_literal_translation: "modismo literal",
  preposition_infinitive: "preposición + infinitivo",
  pronoun_possession: "pronombre/posesión",
  false_friend: "falso amigo",
  lexical_choice: "elección léxica",
  word_order: "orden de palabras",
  tense_agreement: "tiempo/concordancia",
};

export function errorPatternLabel(code: ErrorPatternCode): string {
  return ERROR_PATTERN_LABELS[code];
}

export function mostFrequentPattern(
  patterns: readonly ErrorPatternAggregate[],
): ErrorPatternAggregate | null {
  let top: ErrorPatternAggregate | null = null;
  for (const pattern of patterns) {
    if (top === null || pattern.count > top.count) top = pattern;
  }
  return top;
}
