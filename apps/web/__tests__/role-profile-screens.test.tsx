import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("react-dom", async () => {
  const actual = await vi.importActual<typeof import("react-dom")>("react-dom");
  return { ...actual, useFormStatus: () => ({ pending: false, data: null, method: null, action: null }) };
});

vi.mock("@/features/role-profiles/actions", () => ({
  addSkillForm: vi.fn(),
  createCanonicalSkillForm: vi.fn(),
  removeSkillForm: vi.fn(),
  reorderSkillForm: vi.fn(),
  updateSkillForm: vi.fn(),
  publishRoleProfileForm: vi.fn(),
  submitRoleProfileForm: vi.fn(),
  decideRoleProfileForm: vi.fn(),
  resubmitRoleProfileForm: vi.fn(),
  assignApprovalForm: vi.fn(),
  commentOnRoleProfileForm: vi.fn(),
  retryGenerationForm: vi.fn(),
  startGenerationForm: vi.fn(),
  saveSkillReviewForm: vi.fn(),
  saveSkillOrderForm: vi.fn(),
  mergeSkillsForm: vi.fn(),
}));

vi.mock("@/features/role-profiles/job-refresh", () => ({
  JobRefresh: () => null,
}));

import { GenerationPanel } from "@/features/role-profiles/generation-panel";
import { ReviewPanel } from "@/features/role-profiles/review-panel";
import { Workbench } from "@/features/role-profiles/workbench";
import type { GenerationJob, RoleProfile } from "@/lib/role-profile";

afterEach(() => {
  cleanup();
});

function profile(status: string): RoleProfile {
  return {
    id: "profile-1",
    organisationId: "org-a",
    roleId: "role-1",
    roleName: "Area Sales Manager",
    createdByPersonId: "person-1",
    createdAt: "2026-10-23T00:00:00Z",
    version: {
      id: "version-1",
      profileId: "profile-1",
      organisationId: "org-a",
      versionNumber: 1,
      revision: 3,
      status,
      purpose: "Own regional revenue",
      responsibilities: "Lead the sales cycle",
      businessOutcomes: "Grow revenue",
      decisionScope: "Approve discounts",
      industry: "Business services",
      region: "West Africa",
      teamSize: 8,
      reportingLevel: "manager",
      frameworkVersionId: "",
      sources: [],
      skills: [
        {
          id: "assignment-1",
          skillId: "skill-1",
          skillName: "Territory planning",
          category: "technical",
          definition: "Plans coverage",
          requiredProficiency: 3,
          priority: "required",
          rationale: "The role owns coverage",
          provenance: status === "profile_draft" ? "ai_suggested" : "hr_adjusted",
          position: 1,
        },
      ],
      ai: {
        model: "fake-model",
        sourceContextVersion: "1",
        generatedAt: null,
        draft: { skills: [{ skillId: "skill-1", kind: "technical" }] },
      },
      createdAt: "2026-10-23T00:00:00Z",
    },
  };
}

describe("role profile screens", { timeout: 20_000 }, () => {
  it("shows draft skill controls and the recorded kind", async () => {
    render(<Workbench profile={profile("profile_draft")} />);
    expect(screen.getByRole("heading", { name: "Area Sales Manager" })).toBeInTheDocument();
    expect(screen.getAllByText(/AI suggestion/).length).toBeGreaterThan(0);
    expect(screen.getByText("Recorded kind: technical")).toBeInTheDocument();
    expect(screen.getAllByText("No skills in this category.").length).toBeGreaterThan(0);
    expect(screen.getByRole("button", { name: "Save changes" })).toBeDisabled();
    expect(screen.getByRole("checkbox", { name: "Confirm removal of Territory planning" })).toBeRequired();
    await userEvent.type(screen.getByRole("textbox", { name: "Skill name for Territory planning" }), " updated");
    expect(screen.getByText("Unsaved changes")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save changes" })).toBeEnabled();
  }, 20_000);

  it("does not offer skill editing on a published version", () => {
    render(<Workbench profile={profile("profile_published")} />);
    expect(screen.queryByRole("button", { name: "Save changes" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Remove skill" })).not.toBeInTheDocument();
    expect(screen.getAllByText("profile_published").length).toBeGreaterThan(0);
  });

  it("offers the current approval actions and keeps rejection off a draft", () => {
    const history = { ok: true as const, value: { profileId: "profile-1", events: [], comments: [] } };
    const versions = { ok: true as const, value: [] };
    const approvals = {
      ok: true as const,
      value: {
        assignments: [],
        decisions: [],
        steps: [
          {
            id: "step-1",
            organisationId: "org-a",
            profileVersionId: "version-1",
            stageKey: "leader_approval",
            label: "Business Leader Approval",
            requiredRole: "leader",
            position: 1,
          },
        ],
        currentStage: "leader_approval",
        stageLabel: "Business Leader Approval",
        assigneePersonId: "leader-1",
        outstanding: "Business Leader Approval",
        canDecide: true,
        canResubmit: false,
      },
    };
    const { rerender } = render(<ReviewPanel profile={profile("profile_draft")} history={history} versions={versions} />);
    expect(screen.getByRole("button", { name: "Submit for review" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Approve and save" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Reject" })).not.toBeInTheDocument();
    rerender(<ReviewPanel profile={profile("profile_in_review")} history={history} versions={versions} approvals={approvals} />);
    expect(screen.getByRole("button", { name: "Approve and save" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Request changes" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reject" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Publish" })).not.toBeInTheDocument();
  });

  it("shows a timed-out generation job and a retry control", () => {
    const job: GenerationJob = {
      id: "job-1",
      organisationId: "org-a",
      profileVersionId: "version-1",
      contextVersion: 1,
      contextHash: "abc",
      idempotencyKey: "idem-key-1",
      status: "failed",
      modelId: "fake-model",
      promptVersion: "role-skills-profile-v1",
      inputHash: "in",
      outputHash: "",
      latencyMs: 30,
      promptTokens: 1,
      completionTokens: 0,
      attempts: 3,
      lastError: "timeout",
      unsupported: [],
      uncertainty: [],
      cancelRequested: false,
      createdBy: "person-1",
      createdAt: "2026-10-23T00:00:00Z",
    };
    render(
      <GenerationPanel
        profileId="profile-1"
        versionId="version-1"
        contextVersion={1}
        idempotencyKey="idem-key-1"
        editable
        job={{ ok: true, value: job }}
      />,
    );
    expect(screen.getByTestId("generation-status")).toHaveTextContent("failed");
    expect(screen.getByText("The provider timed out. Retry the same job.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Retry job" })).toBeEnabled();
  });
});
