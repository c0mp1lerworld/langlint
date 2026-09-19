import { QueryClient } from "@tanstack/react-query";

export const DEFAULT_STALE_TIME = 30_000;
export const DEFAULT_GC_TIME = 5 * 60_000;

export function makeQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: DEFAULT_STALE_TIME,
        gcTime: DEFAULT_GC_TIME,
        refetchOnWindowFocus: false,
        retry: 1,
      },
      mutations: {
        retry: 0,
      },
    },
  });
}

let browserQueryClient: QueryClient | undefined;

export function getQueryClient(): QueryClient {
  if (typeof window === "undefined") return makeQueryClient();
  if (browserQueryClient === undefined) browserQueryClient = makeQueryClient();
  return browserQueryClient;
}
