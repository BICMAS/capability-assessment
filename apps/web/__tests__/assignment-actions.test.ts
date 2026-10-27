import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("next/cache", () => ({
  revalidatePath: vi.fn(),
}));

vi.mock("@/lib/session-token", () => ({
  sessionToken: vi.fn(async () => "token"),
}));

vi.mock("@/lib/generated/assessment-plan/plans", () => ({
  VersionNotPublished: class VersionNotPublished extends Error {
    constructor() {
      super("not published");
      this.name = "VersionNotPublished";
    }
  },
  assignPeople: vi.fn(),
}));

import { confirmAssignmentResult } from "@/features/assignments/actions";
import { assignPeople, VersionNotPublished } from "@/lib/generated/assessment-plan/plans";

function form(entries: Record<string, string | string[]>): FormData {
  const data = new FormData();
  for (const [key, value] of Object.entries(entries)) {
    if (Array.isArray(value)) {
      for (const item of value) {
        data.append(key, item);
      }
    } else {
      data.set(key, value);
    }
  }
  return data;
}

describe("confirmAssignmentResult", () => {
  beforeEach(() => {
    vi.mocked(assignPeople).mockReset();
  });

  it("returns validation failure when people are missing", async () => {
    const result = await confirmAssignmentResult(
      form({
        frameworkVersionId: "v1",
        roleId: "r1",
      }),
    );
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.code).toBe("validation");
    }
  });

  it("returns validation failure when a mismatch has no reason", async () => {
    const result = await confirmAssignmentResult(
      form({
        frameworkVersionId: "v1",
        roleId: "r1",
        personId: ["p1"],
        mismatchPersonId: ["p1"],
        exceptionReason: "",
      }),
    );
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.code).toBe("validation");
      expect(result.fields?.exceptionReason).toBeTruthy();
    }
  });

  it("returns success with assignment results", async () => {
    vi.mocked(assignPeople).mockResolvedValue({
      results: [{ personId: "p1", outcome: "assigned", reason: "ok", assignmentId: "a1" }],
    });
    const result = await confirmAssignmentResult(
      form({
        frameworkVersionId: "v1",
        roleId: "r1",
        personId: ["p1"],
      }),
    );
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.data?.results[0]?.assignmentId).toBe("a1");
    }
  });

  it("maps unpublished framework to conflict", async () => {
    vi.mocked(assignPeople).mockRejectedValue(new VersionNotPublished());
    const result = await confirmAssignmentResult(
      form({
        frameworkVersionId: "v1",
        roleId: "r1",
        personId: ["p1"],
      }),
    );
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.code).toBe("conflict");
    }
  });

  it("maps network failure to unavailable", async () => {
    vi.mocked(assignPeople).mockRejectedValue(new TypeError("fetch failed"));
    const result = await confirmAssignmentResult(
      form({
        frameworkVersionId: "v1",
        roleId: "r1",
        personId: ["p1"],
      }),
    );
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.code).toBe("unavailable");
    }
  });
});
