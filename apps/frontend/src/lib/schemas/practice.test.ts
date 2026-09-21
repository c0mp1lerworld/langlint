import { describe, expect, it } from "vitest";
import {
  createPracticeSchema,
  targetRuleSchema,
  updatePracticeSchema,
} from "./practice";

const validInput = {
  source_text: "El perro escapó.",
  draft_text: "The dog escaped.",
  target_rules: [{ verb: "run", tense: "past simple" }],
};

describe("targetRuleSchema", () => {
  it("acepta verbo con tiempo y nota opcionales", () => {
    expect(targetRuleSchema.safeParse({ verb: "run" }).success).toBe(true);
  });

  it("rechaza verbo vacío", () => {
    const result = targetRuleSchema.safeParse({ verb: "" });
    expect(result.success).toBe(false);
  });
});

describe("createPracticeSchema", () => {
  it("acepta una práctica válida", () => {
    const result = createPracticeSchema.safeParse(validInput);
    expect(result.success).toBe(true);
  });

  it("rechaza source vacío", () => {
    expect(createPracticeSchema.safeParse({ ...validInput, source_text: "" }).success).toBe(false);
  });

  it("rechaza draft vacío", () => {
    expect(createPracticeSchema.safeParse({ ...validInput, draft_text: "" }).success).toBe(false);
  });

  it("rechaza target_rules vacío", () => {
    expect(createPracticeSchema.safeParse({ ...validInput, target_rules: [] }).success).toBe(false);
  });

  it("rechaza un texto de más de 2000 caracteres", () => {
    const tooLong = "a".repeat(2001);
    expect(createPracticeSchema.safeParse({ ...validInput, source_text: tooLong }).success).toBe(
      false,
    );
    expect(createPracticeSchema.safeParse({ ...validInput, draft_text: tooLong }).success).toBe(
      false,
    );
  });

  it("rechaza más de 5 reglas objetivo", () => {
    const rules = Array.from({ length: 6 }, () => ({ verb: "run" }));
    expect(createPracticeSchema.safeParse({ ...validInput, target_rules: rules }).success).toBe(
      false,
    );
  });
});

describe("updatePracticeSchema", () => {
  it("acepta un objeto vacío (parcial)", () => {
    expect(updatePracticeSchema.safeParse({}).success).toBe(true);
  });

  it("acepta la actualización de un solo campo", () => {
    expect(updatePracticeSchema.safeParse({ draft_text: "New draft." }).success).toBe(true);
  });
});
