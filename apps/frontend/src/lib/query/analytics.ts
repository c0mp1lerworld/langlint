"use client";

import { useQuery } from "@tanstack/react-query";
import { apiClient } from "@/lib/api/client";
import type { components } from "@/lib/api/gen";

type ErrorPatternStats = components["schemas"]["ErrorPatternStats"];
type ProgressSeries = components["schemas"]["ProgressSeries"];
type QuizStats = components["schemas"]["QuizStats"];
type Window = components["schemas"]["Window"];

export const analyticsKeys = {
  errorPatterns: (window: Window) => ["analytics", "error-patterns", window] as const,
  progress: (window: Window) => ["analytics", "progress", window] as const,
  quiz: () => ["analytics", "quiz"] as const,
};

export function useErrorPatternStats(window: Window) {
  return useQuery({
    queryKey: analyticsKeys.errorPatterns(window),
    queryFn: () =>
      apiClient.get<ErrorPatternStats>("/analytics/error-patterns", { query: { window } }),
  });
}

export function useProgressSeries(window: Window) {
  return useQuery({
    queryKey: analyticsKeys.progress(window),
    queryFn: () => apiClient.get<ProgressSeries>("/analytics/progress", { query: { window } }),
    retry: false,
  });
}

export function useQuizStats() {
  return useQuery({
    queryKey: analyticsKeys.quiz(),
    queryFn: () => apiClient.get<QuizStats>("/analytics/quiz"),
  });
}
