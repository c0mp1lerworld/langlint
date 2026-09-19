import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient, buildPath } from "./client";
import { ApiError } from "./errors";

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  });
}

function makeClient(fetchImpl: typeof fetch, getAuthToken?: () => string | undefined): ApiClient {
  return new ApiClient({ baseUrl: "http://localhost:8080/", fetchImpl, getAuthToken });
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("buildPath", () => {
  it("sustituye y codifica los parámetros", () => {
    expect(buildPath("/practices/{practiceId}", { practiceId: "a b/c" })).toBe("/practices/a%20b%2Fc");
  });
});

describe("ApiClient", () => {
  it("get construye la URL con la base normalizada", async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(jsonResponse({ items: [], total: 0 }));
    await makeClient(fetchMock).get("/practices");

    const [url, init] = fetchMock.mock.calls[0] ?? [];
    expect(url).toBe("http://localhost:8080/practices");
    expect(init?.method).toBe("GET");
  });

  it("get añade query params", async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(jsonResponse({}));
    await makeClient(fetchMock).get("/analytics/error-patterns", { query: { window: "day" } });

    const [url] = fetchMock.mock.calls[0] ?? [];
    expect(String(url)).toBe("http://localhost:8080/analytics/error-patterns?window=day");
  });

  it("post envía JSON y Content-Type", async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(jsonResponse({ id: "1" }, 201));
    await makeClient(fetchMock).post("/practices", { source_text: "a" });

    const [, init] = fetchMock.mock.calls[0] ?? [];
    expect(init?.method).toBe("POST");
    expect(init?.body).toBe(JSON.stringify({ source_text: "a" }));
    expect(new Headers(init?.headers).get("Content-Type")).toBe("application/json");
  });

  it("inyecta el header Authorization si hay token", async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(jsonResponse({}));
    await makeClient(fetchMock, () => "secret-token").get("/practices");

    const [, init] = fetchMock.mock.calls[0] ?? [];
    expect(new Headers(init?.headers).get("Authorization")).toBe("Bearer secret-token");
  });

  it("204 → resuelve undefined", async () => {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 204 }));
    const path = buildPath("/practices/{practiceId}", { practiceId: "abc" });
    await expect(makeClient(fetchMock).delete(path)).resolves.toBeUndefined();
  });

  it("respuesta de error → lanza ApiError con el código del contrato", async () => {
    const fetchMock = vi
      .fn<typeof fetch>()
      .mockResolvedValue(jsonResponse({ code: "validation_error", message: "bad" }, 422));

    await expect(makeClient(fetchMock).get("/practices")).rejects.toMatchObject({
      status: 422,
      code: "validation_error",
    });
  });

  it("fallo de red (fetch rechaza) → ApiError status 0", async () => {
    const fetchMock = vi.fn<typeof fetch>().mockRejectedValue(new TypeError("network down"));
    const error = await makeClient(fetchMock).get("/practices").catch((reason: unknown) => reason);
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).status).toBe(0);
  });

  it("usa el fetch global cuando no se inyecta fetchImpl", async () => {
    const fetchStub = vi.fn<typeof fetch>().mockResolvedValue(jsonResponse({ items: [], total: 0 }));
    vi.stubGlobal("fetch", fetchStub);

    const client = new ApiClient({ baseUrl: "http://localhost:8080" });
    await client.get("/practices");

    expect(fetchStub).toHaveBeenCalledOnce();
    expect(String(fetchStub.mock.calls[0]?.[0])).toBe("http://localhost:8080/practices");
  });
});
