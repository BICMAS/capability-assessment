import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("next/cache", () => ({ revalidatePath: vi.fn() }));
vi.mock("next/navigation", () => ({ redirect: vi.fn() }));
vi.mock("@/lib/session-token", () => ({
  sessionTokenOrThrow: vi.fn(async () => "token"),
}));

vi.mock("@/lib/role-profile", async () => {
  const actual = await vi.importActual<typeof import("@/lib/role-profile")>("@/lib/role-profile");
  return {
    ...actual,
    updateRoleProfileContext: vi.fn(),
    updateRoleProfileSkill: vi.fn(),
    retryRoleProfileJob: vi.fn(),
    publishRoleProfile: vi.fn(),
    submitRoleProfile: vi.fn(),
  };
});

import { AppError } from "@/lib/errors";
import { publishRoleProfile, retryRoleProfileJob, RoleProfileApiError, submitRoleProfile, updateRoleProfileContext, updateRoleProfileSkill } from "@/lib/role-profile";
import { sessionTokenOrThrow } from "@/lib/session-token";

import { publishRoleProfileResult, removeSkillResult, retryGenerationResult, saveRoleContextResult, submitRoleProfileResult, updateSkillResult } from "@/features/role-profiles/actions";

function form(entries: Record<string, string>): FormData {
  const data = new FormData();
  for (const [key, value] of Object.entries(entries)) {
    data.set(key, value);
  }
  return data;
}

const context = {
  profileId: "profile-1",
  versionId: "version-1",
  expectedRevision: "2",
  purpose: "Own regional revenue and customer relationships",
  responsibilities: "Lead the sales cycle from qualification through renewal",
  businessOutcomes: "Grow retained revenue",
  decisionScope: "Approve standard discounts",
  industry: "Business services",
  region: "West Africa",
  teamSize: "8",
  reportingLevel: "manager",
};

beforeEach(() => {
  vi.mocked(sessionTokenOrThrow).mockResolvedValue("token");
  vi.mocked(updateRoleProfileContext).mockReset();
  vi.mocked(updateRoleProfileSkill).mockReset();
  vi.mocked(retryRoleProfileJob).mockReset();
  vi.mocked(publishRoleProfile).mockReset();
  vi.mocked(submitRoleProfile).mockReset();
});

describe("role profile actions", () => {
  it("rejects an invalid context before calling the service", async () => {
    const result = await saveRoleContextResult(form({ ...context, reportingLevel: "boss", teamSize: "0" }));
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.code).toBe("validation");
    }
    expect(updateRoleProfileContext).not.toHaveBeenCalled();
  });

  it("saves context with the revision the form sent", async () => {
    vi.mocked(updateRoleProfileContext).mockResolvedValue({} as never);
    const result = await saveRoleContextResult(form(context));
    expect(result.ok).toBe(true);
    expect(updateRoleProfileContext).toHaveBeenCalledWith(
      "token",
      "version-1",
      expect.objectContaining({ expectedRevision: 2, reportingLevel: "manager", teamSize: 8 }),
    );
  });

  it("does not remove a skill until the confirmation is present", async () => {
    const result = await removeSkillResult(
      form({ profileId: "profile-1", versionId: "version-1", skillAssignmentId: "assignment-1", expectedRevision: "2" }),
    );
    expect(result).toMatchObject({ ok: false, code: "validation" });
  });

  it("returns the service conflict when a skill edit is stale", async () => {
    vi.mocked(updateRoleProfileSkill).mockRejectedValue(new RoleProfileApiError(409, "conflict"));
    const result = await updateSkillResult(
      form({
        profileId: "profile-1",
        versionId: "version-1",
        skillAssignmentId: "assignment-1",
        expectedRevision: "2",
        definition: "Plans the territory",
        requiredProficiency: "3",
        priority: "required",
        rationale: "The role owns coverage",
      }),
    );
    expect(result).toMatchObject({ ok: false, code: "conflict" });
    expect(updateRoleProfileSkill).toHaveBeenCalled();
  });

  it("retries the same generation job", async () => {
    vi.mocked(retryRoleProfileJob).mockResolvedValue({ id: "job-1", status: "queued" } as never);
    const result = await retryGenerationResult(form({ profileId: "profile-1", jobId: "job-1" }));
    expect(result).toMatchObject({ ok: true, data: { jobId: "job-1" } });
  });

  it("submits a draft for review when the service accepts it", async () => {
    vi.mocked(submitRoleProfile).mockResolvedValue({ status: "profile_in_review" } as never);
    const result = await submitRoleProfileResult(form({ profileId: "profile-1", versionId: "version-1", expectedRevision: "4" }));
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.message).toContain("review");
    }
  });

  it("reports a forbidden publish instead of a successful approval", async () => {
    vi.mocked(publishRoleProfile).mockRejectedValue(new RoleProfileApiError(403, "forbidden"));
    const result = await publishRoleProfileResult(form({ profileId: "profile-1", versionId: "version-1", expectedRevision: "4" }));
    expect(result).toMatchObject({ ok: false, code: "forbidden" });
  });

  it("reports a failed submission from the service", async () => {
    vi.mocked(submitRoleProfile).mockRejectedValue(new RoleProfileApiError(409, "immutable"));
    const result = await submitRoleProfileResult(form({ profileId: "profile-1", versionId: "version-1", expectedRevision: "4" }));
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.message).toBe("This version cannot be changed.");
    }
  });

  it("asks the user to sign in again when the session is missing", async () => {
    vi.mocked(sessionTokenOrThrow).mockRejectedValue(new AppError("unauthenticated"));
    const result = await saveRoleContextResult(form(context));
    expect(result).toMatchObject({ ok: false, code: "unauthenticated" });
  });
});
