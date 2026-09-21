"use client";

import { isAnalysisPending, userMessage } from "@/lib/api/errors";
import type { components } from "@/lib/api/gen";
import { useAnalyzePractice, usePractice } from "@/lib/query/practices";
import { FragmentDiff } from "./fragment-diff";
import { PracticeStatusBadge } from "./status-badge";

type Fragment = components["schemas"]["Fragment"];

function Fragments({ fragments, practiceId }: { fragments: Fragment[]; practiceId: string }) {
  if (fragments.length === 0) {
    return <p className="text-gray-600">El análisis no contiene fragmentos.</p>;
  }
  return (
    <ul className="space-y-4" aria-label="Fragmentos del análisis">
      {fragments.map((fragment, index) => (
        <FragmentDiff key={index} fragment={fragment} practiceId={practiceId} index={index} />
      ))}
    </ul>
  );
}

export function PracticeDiff({ id }: { id: string }) {
  const practiceQuery = usePractice(id);
  const analyze = useAnalyzePractice(id);

  const analyzeError =
    analyze.isError && !isAnalysisPending(analyze.error) ? userMessage(analyze.error) : null;
  const practice = practiceQuery.data;

  return (
    <section aria-labelledby="practice-diff-heading" className="space-y-4">
      <header className="flex flex-wrap items-center justify-between gap-2">
        <h1 id="practice-diff-heading" className="text-xl font-semibold text-gray-900">
          Práctica
        </h1>
        {practice !== undefined ? (
          <PracticeStatusBadge status={practice.status} />
        ) : null}
      </header>

      {practiceQuery.isPending ? (
        <p role="status" className="text-gray-600">
          Cargando práctica…
        </p>
      ) : null}

      {practiceQuery.isError ? (
        <p role="alert" className="rounded-md bg-red-50 p-3 text-red-800">
          {userMessage(practiceQuery.error)}
        </p>
      ) : null}

      {analyzeError !== null ? (
        <p role="alert" className="rounded-md bg-red-50 p-3 text-red-800">
          {analyzeError}
        </p>
      ) : null}

      {practice !== undefined && practice.status === "draft" ? (
        <div className="rounded-md border border-gray-200 bg-gray-50 p-4">
          <p className="text-gray-700">Esta práctica todavía no tiene análisis.</p>
          <button
            type="button"
            onClick={() => analyze.mutate()}
            disabled={analyze.isPending}
            className="mt-3 rounded-md bg-blue-700 px-4 py-2 text-sm font-medium text-white hover:bg-blue-800 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600 disabled:opacity-60"
          >
            {analyze.isPending ? "Analizando…" : "Analizar"}
          </button>
        </div>
      ) : null}

      {practice !== undefined && practice.status === "analyzing" ? (
        <p role="status" className="rounded-md bg-blue-50 p-3 text-blue-900">
          Analizando tu borrador. Esta página se actualizará automáticamente.
        </p>
      ) : null}

      {practice !== undefined && practice.status === "failed" ? (
        <p role="alert" className="rounded-md bg-red-50 p-3 text-red-800">
          El análisis no pudo completarse.
        </p>
      ) : null}

      {practice !== undefined && practice.status === "completed" ? (
        <Fragments fragments={practice.analysis?.fragments ?? []} practiceId={id} />
      ) : null}
    </section>
  );
}
