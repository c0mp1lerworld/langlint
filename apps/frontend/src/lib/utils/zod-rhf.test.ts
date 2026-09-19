import { describe, expect, it } from "vitest";
import { createPracticeSchema, type CreatePracticeInput } from "@/lib/schemas/practice";
import { zodResolver } from "./zod-rhf";

interface ResolverOutput {
  values: Record<string, unknown>;
  errors: Record<string, unknown>;
}

async function run(values: unknown): Promise<ResolverOutput> {
  const resolver = zodResolver(createPracticeSchema);
  return (await resolver(
    values as CreatePracticeInput,
    undefined,
    undefined as never,
  )) as unknown as ResolverOutput;
}

describe("zodResolver", () => {
  it("valores válidos → los devuelve y sin errores", async () => {
    const values = { source_text: "a", draft_text: "b", target_rules: [{ verb: "run" }] };
    const { values: parsed, errors } = await run(values);
    expect(parsed).toEqual(values);
    expect(errors).toEqual({});
  });

  it("campo superior inválido → error en su clave", async () => {
    const { errors } = await run({ source_text: "", draft_text: "b", target_rules: [{ verb: "run" }] });
    const sourceError = errors.source_text as { message?: string };
    expect(sourceError.message).toBe("El texto en español es obligatorio");
  });

  it("target_rules vacío → error de campo con root", async () => {
    const { errors } = await run({ source_text: "a", draft_text: "b", target_rules: [] });
    const targetRulesError = errors.target_rules as { message?: string; root?: { message?: string } };
    expect(targetRulesError.message).toBe("Añade al menos una regla objetivo");
    expect(targetRulesError.root?.message).toBe("Añade al menos una regla objetivo");
  });

  it("error anidado → target_rules[0].verb", async () => {
    const { errors } = await run({ source_text: "a", draft_text: "b", target_rules: [{ verb: "" }] });
    const targetRules = errors.target_rules as unknown as Array<{ verb?: { message?: string } }>;
    expect(targetRules[0]?.verb?.message).toBe("El verbo es obligatorio");
  });
});
