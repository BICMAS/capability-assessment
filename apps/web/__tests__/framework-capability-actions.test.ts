import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("next/cache", () => ({
  revalidatePath: vi.fn(),
}));

vi.mock("@/lib/session-token", () => ({
  sessionToken: vi.fn(async () => "token"),
}));

vi.mock("@/lib/capability-framework", () => ({
  addRoleCapability: vi.fn(),
  addCapability: vi.fn(),
  addRequirement: vi.fn(),
}));

import { addFrameworkCapabilityResult } from "@/features/frameworks/actions";
import { addCapability, addRequirement, addRoleCapability } from "@/lib/capability-framework";

function form(entries: Record<string, string>): FormData {
  const data = new FormData();
  for (const [key, value] of Object.entries(entries)) {
    data.set(key, value);
  }
  return data;
}

describe("addFrameworkCapabilityResult", () => {
  beforeEach(() => {
    vi.mocked(addRoleCapability).mockReset();
    vi.mocked(addCapability).mockReset();
    vi.mocked(addRequirement).mockReset();
  });

  it("uses the atomic role-capabilities endpoint", async () => {
    vi.mocked(addRoleCapability).mockResolvedValue({
      capability: { id: "c1", name: "Negotiation", levels: [] },
      requirement: { id: "rq1", roleId: "r1", capabilityId: "c1", requiredLevel: 4 },
    });
    const result = await addFrameworkCapabilityResult(
      form({
        versionId: "v1",
        roleId: "r1",
        capabilityName: "Negotiation",
        requiredLevel: "4",
      }),
    );
    expect(result.ok).toBe(true);
    expect(addRoleCapability).toHaveBeenCalledWith("token", "v1", "r1", "Negotiation", 4);
    expect(addCapability).not.toHaveBeenCalled();
    expect(addRequirement).not.toHaveBeenCalled();
  });

  it("returns validation failure for a bad level", async () => {
    const result = await addFrameworkCapabilityResult(
      form({
        versionId: "v1",
        roleId: "r1",
        capabilityName: "Negotiation",
        requiredLevel: "9",
      }),
    );
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.code).toBe("validation");
    }
  });

  it("maps forbidden backend responses without leaking details", async () => {
    vi.mocked(addRoleCapability).mockRejectedValue(new Error("addRoleCapability failed with status 403: secret"));
    const result = await addFrameworkCapabilityResult(
      form({
        versionId: "v1",
        roleId: "r1",
        capabilityName: "Negotiation",
        requiredLevel: "4",
      }),
    );
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.code).toBe("forbidden");
      expect(result.message).not.toContain("secret");
    }
  });
});
