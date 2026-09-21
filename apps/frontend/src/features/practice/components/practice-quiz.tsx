"use client";

import { useId, useState } from "react";
import type { FormEvent } from "react";
import { userMessage } from "@/lib/api/errors";
import type { components } from "@/lib/api/gen";
import { useEvaluateQuizAnswer, useGenerateQuizQuestion } from "@/lib/query/quiz";

type QuizEvaluation = components["schemas"]["QuizEvaluation"];

const FIELD_CLASS =
  "mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-gray-900 shadow-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600";

const BUTTON_CLASS =
  "rounded-md bg-blue-700 px-4 py-2 text-sm font-medium text-white hover:bg-blue-800 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600 disabled:opacity-60";

// PracticeQuiz is the active-practice loop (PRODUCT_DOMAIN §1.2): it asks the
// AI for a question anchored to the fragment's error, then grades the answer
// and, when useful, follows up. Both calls are on demand (mutations).
export function PracticeQuiz({
  practiceId,
  fragmentIndex,
}: {
  practiceId: string;
  fragmentIndex: number;
}) {
  const inputId = useId();
  const generate = useGenerateQuizQuestion(practiceId);
  const evaluate = useEvaluateQuizAnswer(practiceId);

  const [question, setQuestion] = useState<string | null>(null);
  const [answer, setAnswer] = useState("");
  const [evaluation, setEvaluation] = useState<QuizEvaluation | null>(null);

  function start() {
    setEvaluation(null);
    setAnswer("");
    generate.mutate({ fragmentIndex }, { onSuccess: (generated) => setQuestion(generated.prompt) });
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (question === null) return;
    evaluate.mutate(
      { fragmentIndex, question, answer },
      {
        onSuccess: (result) => {
          setEvaluation(result);
          if (result.follow_up !== "") {
            setQuestion(result.follow_up);
            setAnswer("");
          }
        },
      },
    );
  }

  return (
    <section className="rounded-md bg-white p-3 ring-1 ring-gray-200" aria-label="Ponlo en práctica">
      <h3 className="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500">
        Ponlo en práctica
      </h3>

      {question === null ? (
        <div>
          <p className="text-sm text-gray-700">
            Genera una pregunta sobre tu error para practicarlo.
          </p>
          <button type="button" onClick={start} disabled={generate.isPending} className={`mt-2 ${BUTTON_CLASS}`}>
            {generate.isPending ? "Generando…" : "Ponlo en práctica"}
          </button>
        </div>
      ) : (
        <form onSubmit={submit} className="space-y-3">
          <p className="text-sm text-gray-900">{question}</p>
          <div>
            <label htmlFor={inputId} className="block text-xs font-medium text-gray-700">
              Tu respuesta
            </label>
            <textarea
              id={inputId}
              rows={2}
              value={answer}
              onChange={(event) => setAnswer(event.target.value)}
              className={FIELD_CLASS}
            />
          </div>
          <button type="submit" disabled={evaluate.isPending || answer.trim() === ""} className={BUTTON_CLASS}>
            {evaluate.isPending ? "Evaluando…" : "Responder"}
          </button>
        </form>
      )}

      {generate.isError ? (
        <p role="alert" className="mt-2 rounded-md bg-red-50 p-3 text-sm text-red-800">
          {userMessage(generate.error)}
        </p>
      ) : null}
      {evaluate.isError ? (
        <p role="alert" className="mt-2 rounded-md bg-red-50 p-3 text-sm text-red-800">
          {userMessage(evaluate.error)}
        </p>
      ) : null}

      {evaluation !== null ? (
        <div
          role="status"
          className={`mt-3 rounded-md p-3 text-sm ${
            evaluation.correct ? "bg-green-50 text-green-900" : "bg-amber-50 text-amber-900"
          }`}
        >
          <p className="font-semibold">{evaluation.correct ? "¡Correcto!" : "Revisa esto"}</p>
          <p>{evaluation.feedback}</p>
        </div>
      ) : null}
    </section>
  );
}
