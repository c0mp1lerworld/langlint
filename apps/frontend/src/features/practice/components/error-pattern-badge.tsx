import { Tooltip } from "@/components/ui/tooltip";
import type { components } from "@/lib/api/gen";
import { errorPatternLabel } from "@/lib/utils/error-patterns";

type ErrorPattern = components["schemas"]["ErrorPattern"];

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
  const label = errorPatternLabel(pattern.code);
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
