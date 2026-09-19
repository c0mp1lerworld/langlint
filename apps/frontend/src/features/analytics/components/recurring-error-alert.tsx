import { errorPatternLabel, type ErrorPatternAggregate } from "@/lib/utils/error-patterns";

export function RecurringErrorAlert({ pattern }: { pattern: ErrorPatternAggregate | null }) {
  if (pattern === null) return null;

  const times = pattern.count === 1 ? "vez" : "veces";
  return (
    <p
      role="status"
      className="rounded-md border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900"
    >
      Tu error más frecuente es <strong>{errorPatternLabel(pattern.code)}</strong> ({pattern.count}{" "}
      {times}).
    </p>
  );
}
