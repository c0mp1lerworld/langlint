"use client";

import { useId, useState } from "react";
import { ErrorPatternBadge } from "./error-pattern-badge";
import type { components } from "@/lib/api/gen";
import { diffWords, type DiffToken } from "@/lib/utils/diff";

type Fragment = components["schemas"]["Fragment"];

function DiffText({ tokens, variant }: { tokens: DiffToken[]; variant: "draft" | "correction" }) {
  return (
    <>
      {tokens.map((token, tokenIndex) => {
        if (variant === "draft" && token.type === "ins") return null;
        if (variant === "correction" && token.type === "del") return null;

        let className = "";
        if (token.type === "del") {
          className = "rounded bg-red-50 text-red-700 line-through decoration-red-700 decoration-2";
        } else if (token.type === "ins") {
          className = "rounded bg-green-50 font-semibold text-green-800";
        }

        return (
          <span key={`${token.type}-${tokenIndex}`} className={className}>
            {token.value}
          </span>
        );
      })}
    </>
  );
}

export function FragmentDiff({ fragment }: { fragment: Fragment }) {
  const [open, setOpen] = useState(false);
  const detailsId = useId();
  const tokens = diffWords(fragment.user_draft, fragment.correction);

  return (
    <li className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <section aria-label="Español" className="min-w-0">
          <h3 className="mb-1 text-xs font-semibold uppercase tracking-wide text-gray-500">
            Español
          </h3>
          <p className="whitespace-pre-wrap break-words text-gray-900">{fragment.source_es}</p>
        </section>

        <section aria-label="Tu borrador" className="min-w-0">
          <h3 className="mb-1 text-xs font-semibold uppercase tracking-wide text-gray-500">
            Tu borrador
          </h3>
          <p className="whitespace-pre-wrap break-words text-gray-900">
            <DiffText tokens={tokens} variant="draft" />
          </p>
        </section>

        <section aria-label="Corrección IA" className="min-w-0">
          <h3 className="mb-1 text-xs font-semibold uppercase tracking-wide text-gray-500">
            Corrección IA
          </h3>
          <p className="whitespace-pre-wrap break-words text-gray-900">
            <DiffText tokens={tokens} variant="correction" />
          </p>
        </section>
      </div>

      <div className="mt-3">
        <button
          type="button"
          aria-expanded={open}
          aria-controls={detailsId}
          onClick={() => setOpen((value) => !value)}
          className="rounded text-sm font-medium text-blue-700 underline-offset-2 hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600"
        >
          {open ? "Ocultar explicación" : "Ver explicación"}
        </button>

        <div
          id={detailsId}
          hidden={!open}
          className="mt-2 space-y-2 rounded-md bg-gray-50 p-3 text-sm text-gray-800"
        >
          <p>
            <span className="font-semibold">Verbo objetivo: </span>
            {fragment.target_verb_review}
          </p>
          <p>
            <span className="font-semibold">Aclaración léxica: </span>
            {fragment.lexical_clarification}
          </p>
          <p>
            <span className="font-semibold">Explicación gramatical: </span>
            {fragment.grammar_explanation}
          </p>
          {fragment.error_patterns.length > 0 ? (
            <ul className="flex flex-wrap gap-2 pt-1" aria-label="Patrones de error">
              {fragment.error_patterns.map((pattern, patternIndex) => (
                <li key={`${pattern.code}-${patternIndex}`}>
                  <ErrorPatternBadge pattern={pattern} />
                </li>
              ))}
            </ul>
          ) : null}
        </div>
      </div>
    </li>
  );
}
