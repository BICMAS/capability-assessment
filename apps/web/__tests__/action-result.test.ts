import { describe, expect, it } from "vitest";

import { fail, ok } from "@/lib/action-result";
import { AppError, toActionError, USER_MESSAGES } from "@/lib/errors";
import { parseForm } from "@/lib/form-data";
import { z } from "zod";

describe("ActionResult helpers", () => {
  it("builds success results", () => {
    expect(ok({ id: "1" }, "Saved")).toEqual({ ok: true, data: { id: "1" }, message: "Saved" });
  });

  it("builds validation failures with fields", () => {
    const result = fail("validation", USER_MESSAGES.validation, { name: ["Required"] });
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.code).toBe("validation");
      expect(result.fields?.name).toEqual(["Required"]);
    }
  });
});

describe("toActionError", () => {
  it("maps AppError codes", () => {
    const result = toActionError(new AppError("forbidden"));
    expect(result).toEqual({ ok: false, code: "forbidden", message: USER_MESSAGES.forbidden });
  });

  it("maps HTTP status embedded in errors", () => {
    expect(toActionError(new Error("path failed with status 503")).code).toBe("unavailable");
    expect(toActionError(new Error("path failed with status 401")).code).toBe("unauthenticated");
    expect(toActionError(new Error("path failed with status 403")).code).toBe("forbidden");
    expect(toActionError(new Error("path failed with status 409")).code).toBe("conflict");
  });

  it("does not leak raw backend messages for unexpected errors", () => {
    const result = toActionError(new Error("SELECT * FROM secrets failed: password=hunter2"));
    expect(result.message).toBe(USER_MESSAGES.unexpected);
    expect(result.message).not.toContain("hunter2");
  });
});

describe("parseForm", () => {
  const schema = z.object({
    name: z.string().min(1, "Enter a name."),
  });

  it("returns typed data on success", () => {
    const parsed = parseForm(schema, { name: "Negotiation" });
    expect(parsed).toEqual({ data: { name: "Negotiation" } });
  });

  it("returns a validation ActionResult on failure", () => {
    const parsed = parseForm(schema, { name: "" });
    expect("error" in parsed).toBe(true);
    if ("error" in parsed) {
      expect(parsed.error.code).toBe("validation");
      expect(parsed.error.fields?.name?.[0]).toContain("Enter a name");
    }
  });
});
