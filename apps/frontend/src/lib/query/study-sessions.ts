"use client";

import { useMutation } from "@tanstack/react-query";
import { apiClient } from "@/lib/api/client";
import type { components } from "@/lib/api/gen";

type StudySession = components["schemas"]["StudySession"];
type Window = components["schemas"]["Window"];

export interface GenerateStudySessionInput {
  window: Window;
}

// useCreateStudySession generates (on demand) a personalized study session from
// the learner's aggregated weakness profile. It is a mutation so the LLM call
// never runs on render.
export function useCreateStudySession() {
  return useMutation({
    mutationFn: ({ window }: GenerateStudySessionInput) =>
      apiClient.post<StudySession>("/study-sessions", undefined, { query: { window } }),
  });
}
