import { describe, expect, it } from "vitest";
import {
  DEFAULT_GC_TIME,
  DEFAULT_STALE_TIME,
  getQueryClient,
  makeQueryClient,
} from "./query-client";

describe("makeQueryClient", () => {
  it("aplica los defaults de queries y mutations", () => {
    const options = makeQueryClient().getDefaultOptions();
    expect(options.queries?.staleTime).toBe(DEFAULT_STALE_TIME);
    expect(options.queries?.gcTime).toBe(DEFAULT_GC_TIME);
    expect(options.queries?.refetchOnWindowFocus).toBe(false);
    expect(options.queries?.retry).toBe(1);
    expect(options.mutations?.retry).toBe(0);
  });
});

describe("getQueryClient", () => {
  it("reutiliza el singleton en el navegador", () => {
    expect(getQueryClient()).toBe(getQueryClient());
  });
});
