import { afterEach, describe, expect, it, vi } from "vitest";

import { readRoleProfileJob, retryRoleProfileJob, RoleProfileApiError, roleProfileFailure } from "@/lib/role-profile";

const job = {
  id: "job-1",
  organisationId: "org-a",
  profileVersionId: "version-1",
  contextVersion: 2,
  contextHash: "abc",
  idempotencyKey: "idem-key-1",
  status: "failed",
  modelId: "fake-model",
  promptVersion: "role-skills-profile-v1",
  inputHash: "in",
  outputHash: "out",
  latencyMs: 12,
  promptTokens: 3,
  completionTokens: 4,
  attempts: 1,
  lastError: "timeout",
  unsupported: [],
  uncertainty: [],
  cancelRequested: false,
  createdBy: "person-1",
  createdAt: "2026-10-23T00:00:00Z",
};

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("role profile generation client", () => {
  it("reads job status and retries without a provider credential", async () => {
    const fetchMock = vi.fn(async (url: string) => {
      const retry = String(url).endsWith("/retry");
      return new Response(JSON.stringify({ ...job, status: retry ? "queued" : job.status }), { status: retry ? 202 : 200 });
    });
    vi.stubGlobal("fetch", fetchMock);

    const current = await readRoleProfileJob("session-token", "job 1");
    const retried = await retryRoleProfileJob("session-token", "job 1");

    expect(current.status).toBe("failed");
    expect(retried.status).toBe("queued");
    const serialized = JSON.stringify(fetchMock.mock.calls);
    expect(serialized).toContain("/role-profile-jobs/job%201");
    expect(serialized).toContain("Bearer session-token");
    expect(serialized).not.toContain("ROLE_PROFILE_MODEL_API_KEY");
    expect(serialized).not.toContain("sk-");
  });

  it("maps stale, forbidden, and timeout responses to safe messages", () => {
    expect(roleProfileFailure(new RoleProfileApiError(409, "stale"))?.message).toBe("The role context changed. Refresh and try again.");
    expect(roleProfileFailure(new RoleProfileApiError(403, "forbidden"))?.code).toBe("forbidden");
    expect(roleProfileFailure(new RoleProfileApiError(504, "timeout"))?.message).toContain("timed out");
  });
});
