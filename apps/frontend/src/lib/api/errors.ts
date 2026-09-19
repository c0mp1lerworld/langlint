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
