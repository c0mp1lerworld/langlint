import type { components } from "./gen";

export type ErrorCode = components["schemas"]["ErrorResponse"]["code"];

const IDEMPOTENT_SUCCESS_CODES: ReadonlySet<ErrorCode> = new Set<ErrorCode>(["analysis_pending"]);

export interface ApiErrorDetails {
  status: number;
  code?: ErrorCode;
  message: string;
}

export class ApiError extends Error {
  readonly status: number;
  readonly code: ErrorCode | undefined;
  readonly isIdempotentSuccess: boolean;

  constructor(details: ApiErrorDetails) {
    super(details.message);
    this.name = "ApiError";
    this.status = details.status;
    this.code = details.code;
    this.isIdempotentSuccess = isIdempotentSuccess(details.status, details.code);
  }
}

export function isIdempotentSuccess(status: number, code?: ErrorCode): boolean {
  return status === 409 && code !== undefined && IDEMPOTENT_SUCCESS_CODES.has(code);
}

export function isAnalysisPending(error: unknown): error is ApiError {
  return error instanceof ApiError && error.code === "analysis_pending";
}

function isErrorResponse(value: unknown): value is components["schemas"]["ErrorResponse"] {
  if (typeof value !== "object" || value === null) return false;
  const candidate = value as { code?: unknown; message?: unknown };
  return typeof candidate.code === "string" && typeof candidate.message === "string";
}

export async function toApiError(response: Response): Promise<ApiError> {
  let code: ErrorCode | undefined;
  let message = response.statusText;

  try {
    const body: unknown = await response.json();
    if (isErrorResponse(body)) {
      code = body.code;
      message = body.message;
    }
  } catch {
    // Non-JSON body: fall back to the status text (F5: never surface raw infra details).
  }

  return new ApiError({ status: response.status, code, message });
}

export function networkError(): ApiError {
  return new ApiError({ status: 0, message: "network_error" });
}

const ERROR_MESSAGES: Record<ErrorCode, string> = {
  validation_error: "Revisa los campos del formulario.",
  not_found: "No encontramos este recurso.",
  analysis_pending: "El análisis ya está en curso.",
  analysis_failed: "El análisis no pudo completarse.",
  invalid_state: "La práctica no está en un estado válido para esta acción.",
  llm_unavailable: "El motor de análisis no responde. Inténtalo de nuevo.",
  llm_output_truncated: "El análisis generó una respuesta demasiado larga. Inténtalo con un texto más corto.",
  not_implemented: "Esta función aún no está disponible.",
  internal: "Ocurrió un error inesperado.",
};

export function userMessage(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 0) return "No hay conexión con el servidor.";
    if (error.code !== undefined) return ERROR_MESSAGES[error.code];
  }
  return "Ocurrió un error inesperado.";
}
