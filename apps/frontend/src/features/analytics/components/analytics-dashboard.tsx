"use client";

import { ApiError, userMessage } from "@/lib/api/errors";
import { useErrorPatternStats, useProgressSeries } from "@/lib/query/analytics";
import { useAnalyticsStore } from "@/lib/store/analytics";
import { mostFrequentPattern } from "@/lib/utils/error-patterns";
import { ErrorPatternFrequency } from "./error-pattern-frequency";
import { RecurringErrorAlert } from "./recurring-error-alert";

const WINDOWS = [
  { value: "day", label: "Día" },
  { value: "week", label: "Semana" },
  { value: "month", label: "Mes" },
] as const;

const WINDOW_BUTTON_CLASSES =
  "px-3 py-1.5 text-sm font-medium focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-blue-600";

function WindowSelector() {
  const activeWindow = useAnalyticsStore((state) => state.window);
  const setWindow = useAnalyticsStore((state) => state.setWindow);

  return (
    <div
      role="group"
      aria-label="Ventana temporal"
      className="inline-flex overflow-hidden rounded-md border border-gray-300"
    >
      {WINDOWS.map((option) => {
        const active = activeWindow === option.value;
        return (
          <button
            key={option.value}
            type="button"
            aria-pressed={active}
            onClick={() => setWindow(option.value)}
            className={`${WINDOW_BUTTON_CLASSES} ${
              active ? "bg-blue-700 text-white" : "bg-white text-gray-700 hover:bg-gray-50"
            }`}
          >
            {option.label}
          </button>
        );
      })}
    </div>
  );
}

function ErrorPatternsSection() {
  const activeWindow = useAnalyticsStore((state) => state.window);
  const stats = useErrorPatternStats(activeWindow);

  if (stats.isPending) {
    return (
      <p role="status" className="text-gray-600">
        Cargando patrones de error…
      </p>
    );
  }

  if (stats.isError) {
    return (
      <p role="alert" className="rounded-md bg-red-50 p-3 text-red-800">
        {userMessage(stats.error)}
      </p>
    );
  }

  return (
    <div className="space-y-3">
      <RecurringErrorAlert pattern={mostFrequentPattern(stats.data.patterns)} />
      <ErrorPatternFrequency patterns={stats.data.patterns} />
    </div>
  );
}

function ProgressSection() {
  const activeWindow = useAnalyticsStore((state) => state.window);
  const progress = useProgressSeries(activeWindow);

  if (progress.isPending) {
    return (
      <p role="status" className="text-gray-600">
        Cargando progreso…
      </p>
    );
  }

  if (progress.isError) {
    if (progress.error instanceof ApiError && progress.error.code === "not_implemented") {
      return (
        <p className="text-gray-600">
          La serie de progreso estará disponible próximamente (Fase 6).
        </p>
      );
    }
    return (
      <p role="alert" className="rounded-md bg-red-50 p-3 text-red-800">
        {userMessage(progress.error)}
      </p>
    );
  }

  if (progress.data.points.length === 0) {
    return <p className="text-gray-600">Todavía no hay datos de progreso en esta ventana.</p>;
  }

  return (
    <ul className="space-y-2" aria-label="Serie temporal de progreso">
      {progress.data.points.map((point) => (
        <li
          key={point.period_start}
          className="flex flex-wrap items-baseline justify-between gap-2 text-sm"
        >
          <span className="text-gray-700">
            {new Date(point.period_start).toLocaleDateString("es-ES")}
          </span>
          <span className="text-gray-900">
            {Math.round(point.accuracy * 100)}% precisión · {point.error_count} errores en{" "}
            {point.total_fragments} fragmentos
          </span>
        </li>
      ))}
    </ul>
  );
}

export function AnalyticsDashboard() {
  return (
    <section aria-labelledby="analytics-heading" className="space-y-6">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 id="analytics-heading" className="text-xl font-semibold text-gray-900">
            Analíticas
          </h1>
          <p className="text-gray-600">Errores recurrentes y progreso de tus prácticas.</p>
        </div>
        <WindowSelector />
      </header>

      <div className="space-y-3">
        <h2 className="text-lg font-semibold text-gray-900">Errores por patrón</h2>
        <ErrorPatternsSection />
      </div>

      <div className="space-y-3">
        <h2 className="text-lg font-semibold text-gray-900">Progreso</h2>
        <ProgressSection />
      </div>
    </section>
  );
}
