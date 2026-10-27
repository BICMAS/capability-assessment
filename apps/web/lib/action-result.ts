export type FieldErrors = Record<string, string[]>;

export type ActionOk<T = undefined> = {
  ok: true;
  data?: T;
  message?: string;
};

export type ActionErr = {
  ok: false;
  code: ActionErrorCode;
  message: string;
  fields?: FieldErrors;
};

export type ActionResult<T = undefined> = ActionOk<T> | ActionErr;

export type ActionErrorCode =
  | "validation"
  | "unauthenticated"
  | "forbidden"
  | "conflict"
  | "not_found"
  | "unavailable"
  | "unexpected";

export function ok<T = undefined>(data?: T, message?: string): ActionOk<T> {
  return { ok: true, data, message };
}

export function fail(code: ActionErrorCode, message: string, fields?: FieldErrors): ActionErr {
  return { ok: false, code, message, fields };
}
