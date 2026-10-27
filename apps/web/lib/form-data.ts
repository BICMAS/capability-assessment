import { z } from "zod";

import { fail, type ActionErr, type FieldErrors } from "@/lib/action-result";
import { USER_MESSAGES } from "@/lib/errors";

export function text(formData: FormData, name: string): string {
  const value = formData.get(name);
  return typeof value === "string" ? value.trim() : "";
}

export function texts(formData: FormData, name: string): string[] {
  return formData.getAll(name).filter((value): value is string => typeof value === "string" && value.trim().length > 0);
}

export function indexed(formData: FormData, name: string): string[] {
  const values: string[] = [];
  for (const [key, value] of formData.entries()) {
    if ((key === name || key.startsWith(`${name}.`)) && typeof value === "string" && value.trim().length > 0) {
      values.push(value.trim());
    }
  }
  return values;
}

export function parseForm<T>(schema: z.ZodType<T>, input: unknown): { data: T } | { error: ActionErr } {
  const parsed = schema.safeParse(input);
  if (parsed.success) {
    return { data: parsed.data };
  }
  const fields: FieldErrors = {};
  for (const issue of parsed.error.issues) {
    const key = issue.path.length > 0 ? issue.path.join(".") : "_form";
    fields[key] = [...(fields[key] ?? []), issue.message];
  }
  return { error: fail("validation", USER_MESSAGES.validation, fields) };
}
