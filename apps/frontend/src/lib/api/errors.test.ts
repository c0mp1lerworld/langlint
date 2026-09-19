import { describe, expect, it } from "vitest";
import {
  ApiError,
  isAnalysisPending,
  isIdempotentSuccess,
  networkError,
  toApiError,
  userMessage,
} from "./errors";

function jsonResponse(body: unknown, status: number): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  });
}

describe("isIdempotentSuccess", () => {
  it("409 analysis_pending → éxito idempotente", () => {
    expect(isIdempotentSuccess(409, "analysis_pending")).toBe(true);
  });

  it("409 invalid_state → no idempotente", () => {
    expect(isIdempotentSuccess(409, "invalid_state")).toBe(false);
  });

  it("200 → no idempotente", () => {
    expect(isIdempotentSuccess(200, "analysis_pending")).toBe(false);
  });
});

describe("ApiError", () => {
  it("marca isIdempotentSuccess en su construcción", () => {
    const error = new ApiError({ status: 409, code: "analysis_pending", message: "pending" });
    expect(error.isIdempotentSuccess).toBe(true);
  });

  it("isAnalysisPending reconoce el código", () => {
    expect(isAnalysisPending(new ApiError({ status: 409, code: "analysis_pending", message: "x" }))).toBe(true);
    expect(isAnalysisPending(new Error("otros"))).toBe(false);
  });
});

describe("toApiError", () => {
  it("parsea un ErrorResponse del contrato", async () => {
    const error = await toApiError(
      jsonResponse({ code: "not_found", message: "No encontramos este recurso." }, 404),
    );
    expect(error.status).toBe(404);
    expect(error.code).toBe("not_found");
    expect(error.message).toBe("No encontramos este recurso.");
  });

  it("body no JSON → cae al statusText sin filtrar detalle", async () => {
    const error = await toApiError(new Response("boom", { status: 500, statusText: "Internal Server Error" }));
    expect(error.code).toBeUndefined();
    expect(error.message).toBe("Internal Server Error");
    expect(error.message).not.toContain("boom");
  });
});

describe("networkError", () => {
  it("representa un fallo de red con status 0", () => {
    const error = networkError();
    expect(error.status).toBe(0);
    expect(error.code).toBeUndefined();
  });
});

describe("userMessage", () => {
  it("status 0 → mensaje de conexión", () => {
    expect(userMessage(networkError())).toBe("No hay conexión con el servidor.");
  });

  it("código conocido → mensaje de UX", () => {
    expect(userMessage(new ApiError({ status: 503, code: "llm_unavailable", message: "x" }))).toBe(
      "El motor de análisis no responde. Inténtalo de nuevo.",
    );
  });

  it("error desconocido → mensaje genérico", () => {
    expect(userMessage(new Error("boom"))).toBe("Ocurrió un error inesperado.");
  });
});
