import { Tooltip } from "@/components/ui/tooltip";
import type { components } from "@/lib/api/gen";

type ErrorPattern = components["schemas"]["ErrorPattern"];

const CODE_LABELS: Record<ErrorPattern["code"], string> = {
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

const SEVERITY_LABELS: Record<ErrorPattern["severity"], string> = {
  minor: "leve",
  moderate: "moderado",
  critical: "crítico",
};

const SEVERITY_CLASSES: Record<ErrorPattern["severity"], string> = {
  minor: "bg-amber-100 text-amber-900",
  moderate: "bg-orange-100 text-orange-900",
  critical: "bg-red-100 text-red-900",
};

export function ErrorPatternBadge({ pattern }: { pattern: ErrorPattern }) {
  const label = CODE_LABELS[pattern.code];
  const severity = SEVERITY_LABELS[pattern.severity];
  const description =
    pattern.note !== undefined && pattern.note.length > 0
      ? `${label} (${severity}): ${pattern.note}`
      : `${label} (${severity})`;

  return (
    <Tooltip label={description}>
      <span
        className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${SEVERITY_CLASSES[pattern.severity]}`}
      >
        {label}
        <span className="sr-only">, severidad {severity}</span>
      </span>
    </Tooltip>
  );
}
