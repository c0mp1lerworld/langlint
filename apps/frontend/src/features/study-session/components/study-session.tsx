"use client";

import { useState } from "react";
import { userMessage } from "@/lib/api/errors";
import type { components } from "@/lib/api/gen";
import { useCreateStudySession } from "@/lib/query/study-sessions";

type StudySession = components["schemas"]["StudySession"];
type Window = components["schemas"]["Window"];

const WINDOWS = [
  { value: "day", label: "Día" },
  { value: "week", label: "Semana" },
  { value: "month", label: "Mes" },
] as const;

const BUTTON_CLASS =
  "rounded-md bg-blue-700 px-4 py-2 text-sm font-medium text-white hover:bg-blue-800 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600 disabled:opacity-60";

const WINDOW_BUTTON_CLASS =
  "px-3 py-1.5 text-sm font-medium focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-blue-600";

// StudySession renders the adaptive tutor (PRODUCT_DOMAIN §12.1): it asks the
// API to generate a personalized study session from the learner's aggregated
// weakness profile and renders the returned theory, traps and exercises. It
// holds no business logic (F11): it only reflects what the contract exposes.
export function StudySession() {
  const create = useCreateStudySession();
  const [window, setWindow] = useState<Window>("week");
  const [session, setSession] = useState<StudySession | null>(null);

  function generate() {
    setSession(null);
    create.mutate({ window }, { onSuccess: setSession });
  }

  return (
    <section aria-labelledby="study-session-heading" className="space-y-6">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 id="study-session-heading" className="text-xl font-semibold text-gray-900">
            Tutor adaptativo
          </h1>
          <p className="text-gray-600">
            Genera una sesión de estudio desde tus puntos débiles recurrentes.
          </p>
        </div>
      </header>

      <div className="space-y-3">
        <div
          role="group"
          aria-label="Ventana temporal"
          className="inline-flex overflow-hidden rounded-md border border-gray-300"
        >
          {WINDOWS.map((option) => {
            const active = window === option.value;
            return (
              <button
                key={option.value}
                type="button"
                aria-pressed={active}
                onClick={() => setWindow(option.value)}
                className={`${WINDOW_BUTTON_CLASS} ${
                  active ? "bg-blue-700 text-white" : "bg-white text-gray-700 hover:bg-gray-50"
                }`}
              >
                {option.label}
              </button>
            );
          })}
        </div>

        <div>
          <button type="button" onClick={generate} disabled={create.isPending} className={BUTTON_CLASS}>
            {create.isPending ? "Generando…" : "Generar sesión de estudio"}
          </button>
        </div>
      </div>

      {create.isError ? (
        <p role="alert" className="rounded-md bg-red-50 p-3 text-red-800">
          {userMessage(create.error)}
        </p>
      ) : null}

      {session !== null ? <SessionContent session={session} /> : null}
    </section>
  );
}

function SessionContent({ session }: { session: StudySession }) {
  return (
    <div className="space-y-4" role="status">
      <div className="rounded-md bg-white p-4 ring-1 ring-gray-200">
        <h2 className="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500">Teoría</h2>
        <p className="text-sm text-gray-900">{session.theory}</p>
      </div>

      {session.traps.length > 0 ? (
        <div className="rounded-md bg-white p-4 ring-1 ring-gray-200">
          <h2 className="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500">
            Trampas comunes
          </h2>
          <ul className="space-y-2">
            {session.traps.map((trap, index) => (
              <li key={`${trap.code}-${index}`} className="text-sm text-gray-900">
                <span className="font-medium">{trap.code}:</span> {trap.description}
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      {session.exercises.length > 0 ? (
        <div className="rounded-md bg-white p-4 ring-1 ring-gray-200">
          <h2 className="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500">
            Ejercicios
          </h2>
          <ul className="space-y-3">
            {session.exercises.map((exercise, index) => (
              <li key={index} className="text-sm">
                <p className="text-gray-900">{exercise.prompt}</p>
                <p className="mt-1 text-gray-600">
                  <span className="font-medium">Respuesta:</span> {exercise.answer}
                </p>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </div>
  );
}
