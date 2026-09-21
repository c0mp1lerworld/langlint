"use client";

import { useId, useState } from "react";
import type { ReactNode } from "react";
import { ErrorPatternBadge } from "./error-pattern-badge";
import { PracticeQuiz } from "./practice-quiz";
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

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="grid grid-cols-[7.5rem_1fr] gap-2">
      <dt className="font-semibold text-gray-700">{label}</dt>
      <dd className="min-w-0 text-gray-800">{children}</dd>
    </div>
  );
}

function Alternatives({ items }: { items: string[] }) {
  if (items.length === 0) {
    return <span className="text-gray-500">—</span>;
  }
  return (
    <ul className="flex flex-wrap gap-1">
      {items.map((item, index) => (
        <li
          key={`${item}-${index}`}
          className="rounded bg-white px-2 py-0.5 text-xs text-gray-700 ring-1 ring-gray-200"
        >
          {item}
        </li>
      ))}
    </ul>
  );
}

function ExplanationCard({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="rounded-md bg-white p-3 ring-1 ring-gray-200">
      <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500">{title}</h3>
      <dl className="space-y-1.5 text-sm">{children}</dl>
    </section>
  );
}

export function FragmentDiff({
  fragment,
  practiceId,
  index,
}: {
  fragment: Fragment;
  practiceId?: string;
  index?: number;
}) {
  const [open, setOpen] = useState(false);
  const detailsId = useId();
  const tokens = diffWords(fragment.user_draft, fragment.correction);
  const verb = fragment.target_verb_review;
  const lexical = fragment.lexical_clarification;
  const grammar = fragment.grammar_explanation;

  return (
    <li className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <section aria-label="Español" className="min-w-0">
          <h2 className="mb-1 text-xs font-semibold uppercase tracking-wide text-gray-500">
            Español
          </h2>
          <p className="whitespace-pre-wrap break-words text-gray-900">{fragment.source_es}</p>
        </section>

        <section aria-label="Tu borrador" className="min-w-0">
          <h2 className="mb-1 text-xs font-semibold uppercase tracking-wide text-gray-500">
            Tu borrador
          </h2>
          <p className="whitespace-pre-wrap break-words text-gray-900">
            <DiffText tokens={tokens} variant="draft" />
          </p>
        </section>

        <section aria-label="Corrección IA" className="min-w-0">
          <h2 className="mb-1 text-xs font-semibold uppercase tracking-wide text-gray-500">
            Corrección IA
          </h2>
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

        <div id={detailsId} hidden={!open} className="mt-2 space-y-2 rounded-md bg-gray-50 p-3">
          <ExplanationCard title="Verbo objetivo">
            <Field label="Verbo">{verb.verb}</Field>
            <Field label="Forma correcta">{verb.correct_form}</Field>
            <Field label="Regla">{verb.rule}</Field>
            <Field label="Por qué">{verb.why}</Field>
            <Field label="En español">{verb.es_contrast}</Field>
            <Field label="Alternativas">
              <Alternatives items={verb.alternatives} />
            </Field>
          </ExplanationCard>

          <ExplanationCard title="Aclaración léxica">
            <Field label="Término">{lexical.term}</Field>
            <Field label="Significado">{lexical.meaning}</Field>
            <Field label="Por qué no">{lexical.why_wrong}</Field>
            <Field label="Alternativas">
              <Alternatives items={lexical.alternatives} />
            </Field>
          </ExplanationCard>

          <ExplanationCard title="Explicación gramatical">
            <Field label="Regla">{grammar.rule_name}</Field>
            <Field label="Cómo se forma">{grammar.construction}</Field>
            <Field label="Por qué">{grammar.explanation}</Field>
            <Field label="Contra-ejemplo">{grammar.counterexample}</Field>
            <Field label="Excepción">{grammar.exception}</Field>
            <Field label="En español">{grammar.es_contrast}</Field>
          </ExplanationCard>

          {fragment.error_patterns.length > 0 ? (
            <section className="rounded-md bg-white p-3 ring-1 ring-gray-200">
              <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500">
                Patrones de error
              </h3>
              <ul className="flex flex-wrap gap-2" aria-label="Patrones de error">
                {fragment.error_patterns.map((pattern, patternIndex) => (
                  <li key={`${pattern.code}-${patternIndex}`}>
                    <ErrorPatternBadge pattern={pattern} />
                  </li>
                ))}
              </ul>
            </section>
          ) : null}

          {practiceId !== undefined && index !== undefined ? (
            <PracticeQuiz practiceId={practiceId} fragmentIndex={index} />
          ) : null}
        </div>
      </div>
    </li>
  );
}
