"use client";

import { useMutation } from "@tanstack/react-query";
import { apiClient, buildPath } from "@/lib/api/client";
import type { components } from "@/lib/api/gen";

type QuizQuestion = components["schemas"]["QuizQuestion"];
type QuizEvaluation = components["schemas"]["QuizEvaluation"];

export interface GenerateQuestionInput {
  fragmentIndex: number;
}

export interface EvaluateAnswerInput {
  fragmentIndex: number;
  question: string;
  answer: string;
}

// useGenerateQuizQuestion generates (on demand) a practice question anchored to
// a fragment's error. It is a mutation so the LLM call never runs on render.
export function useGenerateQuizQuestion(practiceId: string) {
  return useMutation({
    mutationFn: ({ fragmentIndex }: GenerateQuestionInput) =>
      apiClient.post<QuizQuestion>(
        buildPath("/practices/{practiceId}/quiz", { practiceId }),
        { fragment_index: fragmentIndex },
      ),
  });
}

// useEvaluateQuizAnswer grades the learner's answer and may return a follow-up.
export function useEvaluateQuizAnswer(practiceId: string) {
  return useMutation({
    mutationFn: ({ fragmentIndex, question, answer }: EvaluateAnswerInput) =>
      apiClient.post<QuizEvaluation>(
        buildPath("/practices/{practiceId}/quiz/answer", { practiceId }),
        { fragment_index: fragmentIndex, question, answer },
      ),
  });
}
