import { describe, expect, it } from "vitest";
import {
  ERROR_PATTERN_LABELS,
  errorPatternLabel,
  mostFrequentPattern,
  type ErrorPatternAggregate,
} from "./error-patterns";

function aggregate(code: string, count: number): ErrorPatternAggregate {
  return { code: code as ErrorPatternAggregate["code"], count, last_seen_at: "2026-09-19T00:00:00Z" };
}

describe("ERROR_PATTERN_LABELS", () => {
  it("cubre toda la taxonomía del contrato", () => {
    expect(Object.keys(ERROR_PATTERN_LABELS).sort()).toEqual(
      [
        "false_friend",
        "idiom_literal_translation",
        "infinitive_conjugation",
        "lexical_choice",
        "passive_voice_misuse",
        "preposition_infinitive",
        "pronoun_possession",
        "tense_agreement",
        "word_order",
      ].sort(),
    );
  });

  it("errorPatternLabel devuelve la etiqueta del código", () => {
    expect(errorPatternLabel("tense_agreement")).toBe("tiempo/concordancia");
  });
});

describe("mostFrequentPattern", () => {
  it("lista vacía → null", () => {
    expect(mostFrequentPattern([])).toBeNull();
  });

  it("devuelve el de mayor count", () => {
    const patterns = [aggregate("word_order", 1), aggregate("tense_agreement", 3)];
    expect(mostFrequentPattern(patterns)?.code).toBe("tense_agreement");
  });

  it("empate → conserva el primero", () => {
    const patterns = [aggregate("word_order", 2), aggregate("tense_agreement", 2)];
    expect(mostFrequentPattern(patterns)?.code).toBe("word_order");
  });

  it("un solo patrón → ese", () => {
    expect(mostFrequentPattern([aggregate("false_friend", 5)])?.count).toBe(5);
  });
});
