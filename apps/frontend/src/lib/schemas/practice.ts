import { z } from "zod";
import type { components } from "@/lib/api/gen";

type Schemas = components["schemas"];

type AssertAssignable<Actual extends Expected, Expected> = Actual;

export const targetRuleSchema = z.object({
  verb: z.string().min(1, "El verbo es obligatorio"),
  tense: z.string().optional(),
  note: z.string().optional(),
});

export const createPracticeSchema = z.object({
  source_text: z.string().min(1, "El texto en español es obligatorio"),
  draft_text: z.string().min(1, "El borrador en inglés es obligatorio"),
  target_rules: z.array(targetRuleSchema).min(1, "Añade al menos una regla objetivo"),
});

export const updatePracticeSchema = createPracticeSchema.partial();

export type TargetRuleInput = z.infer<typeof targetRuleSchema>;
export type CreatePracticeInput = z.infer<typeof createPracticeSchema>;
export type UpdatePracticeInput = z.infer<typeof updatePracticeSchema>;

export type TargetRuleContract = AssertAssignable<TargetRuleInput, Schemas["TargetRule"]>;
export type CreatePracticeContract = AssertAssignable<
  CreatePracticeInput,
  Schemas["CreatePracticeRequest"]
>;
export type UpdatePracticeContract = AssertAssignable<
  UpdatePracticeInput,
  Schemas["UpdatePracticeRequest"]
>;
