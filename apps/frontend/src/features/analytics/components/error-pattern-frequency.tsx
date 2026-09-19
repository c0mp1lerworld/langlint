import { errorPatternLabel, type ErrorPatternAggregate } from "@/lib/utils/error-patterns";

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString("es-ES", {
    day: "2-digit",
    month: "short",
  });
}

export function ErrorPatternFrequency({ patterns }: { patterns: ErrorPatternAggregate[] }) {
  if (patterns.length === 0) {
    return <p className="text-gray-600">Todavía no hay errores registrados en esta ventana.</p>;
  }

  const maxCount = Math.max(...patterns.map((pattern) => pattern.count), 1);
  const ordered = [...patterns].sort((a, b) => b.count - a.count);

  return (
    <ul className="space-y-3" aria-label="Frecuencia de patrones de error">
      {ordered.map((pattern) => {
        const percent = Math.round((pattern.count / maxCount) * 100);
        const times = pattern.count === 1 ? "vez" : "veces";
        return (
          <li key={pattern.code}>
            <div className="flex flex-wrap items-baseline justify-between gap-2 text-sm">
              <span className="font-medium text-gray-900">{errorPatternLabel(pattern.code)}</span>
              <span className="text-gray-700">
                {pattern.count} {times}
              </span>
            </div>
            <div className="mt-1 h-2 w-full rounded bg-gray-100" aria-hidden="true">
              <div className="h-2 rounded bg-blue-700" style={{ width: `${percent}%` }} />
            </div>
            <p className="mt-0.5 text-xs text-gray-500">Última vez: {formatDate(pattern.last_seen_at)}</p>
          </li>
        );
      })}
    </ul>
  );
}
