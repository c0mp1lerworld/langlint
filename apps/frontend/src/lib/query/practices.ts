"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiClient, buildPath } from "@/lib/api/client";
import { isAnalysisPending } from "@/lib/api/errors";
import type { components } from "@/lib/api/gen";

type PracticeDetail = components["schemas"]["PracticeDetail"];
type PracticeList = components["schemas"]["PracticeList"];

const POLL_INTERVAL_MS = 2_000;

export const practiceKeys = {
  list: () => ["practices"] as const,
  detail: (id: string) => ["practices", id] as const,
};

export function useListPractices() {
  return useQuery({
    queryKey: practiceKeys.list(),
    queryFn: () => apiClient.get<PracticeList>("/practices"),
  });
}

export function usePractice(id: string) {
  return useQuery({
    queryKey: practiceKeys.detail(id),
    queryFn: () =>
      apiClient.get<PracticeDetail>(buildPath("/practices/{practiceId}", { practiceId: id })),
    refetchInterval: (query) =>
      query.state.data?.status === "analyzing" ? POLL_INTERVAL_MS : false,
  });
}

export function useAnalyzePractice(id: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () =>
      apiClient.post<void>(buildPath("/practices/{practiceId}/analyze", { practiceId: id })),
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: practiceKeys.detail(id) });
      const previous = queryClient.getQueryData<PracticeDetail>(practiceKeys.detail(id));
      if (previous !== undefined) {
        queryClient.setQueryData<PracticeDetail>(practiceKeys.detail(id), {
          ...previous,
          status: "analyzing",
        });
      }
      return { previous };
    },
    onError: (error, _variables, context) => {
      if (isAnalysisPending(error)) {
        queryClient.invalidateQueries({ queryKey: practiceKeys.detail(id) });
        return;
      }
      if (context?.previous !== undefined) {
        queryClient.setQueryData<PracticeDetail>(practiceKeys.detail(id), context.previous);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: practiceKeys.detail(id) });
    },
  });
}
