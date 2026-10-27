import { fail, type ActionErr, type ActionErrorCode } from "@/lib/action-result";

/** Safe user-facing messages. Never include raw backend payloads. */
export const USER_MESSAGES = {
  unauthenticated: "Sign in again to continue.",
  forbidden: "You do not have permission to do that.",
  validation: "Check the highlighted fields and try again.",
  conflict: "That record changed. Refresh and try again.",
  not_found: "That record was not found.",
  unavailable: "A required service is unavailable. Try again shortly.",
  unexpected: "Something went wrong. Try again.",
} as const;

export class AppError extends Error {
  readonly code: ActionErrorCode;

  constructor(code: ActionErrorCode, message?: string) {
    super(message ?? USER_MESSAGES[code]);
    this.name = "AppError";
    this.code = code;
  }
}

export function toActionError(error: unknown, fallback: ActionErrorCode = "unexpected"): ActionErr {
  if (error instanceof AppError) {
    return fail(error.code, error.message);
  }
  if (typeof error === "object" && error !== null && "name" in error) {
    const name = String((error as { name?: string }).name ?? "");
    if (name === "PlanForbidden" || name.includes("Forbidden")) {
      return fail("forbidden", USER_MESSAGES.forbidden);
    }
    if (name === "PlanImmutable" || name === "PlanAlreadyGenerated" || name === "VersionNotPublished") {
      return fail("conflict", error instanceof Error ? error.message : USER_MESSAGES.conflict);
    }
  }
  if (error instanceof TypeError) {
    return fail("unavailable", USER_MESSAGES.unavailable);
  }
  if (error instanceof Error) {
    const status = statusFromMessage(error.message);
    if (status === 401) {
      return fail("unauthenticated", USER_MESSAGES.unauthenticated);
    }
    if (status === 403) {
      return fail("forbidden", USER_MESSAGES.forbidden);
    }
    if (status === 404) {
      return fail("not_found", USER_MESSAGES.not_found);
    }
    if (status === 409) {
      return fail("conflict", USER_MESSAGES.conflict);
    }
    if (status === 502 || status === 503 || status === 504) {
      return fail("unavailable", USER_MESSAGES.unavailable);
    }
  }
  return fail(fallback, USER_MESSAGES[fallback]);
}

function statusFromMessage(message: string): number | null {
  const match = message.match(/status\s+(\d{3})/i) ?? message.match(/failed with status (\d{3})/i);
  if (!match) {
    return null;
  }
  return Number(match[1]);
}
