import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { SkillsWorkspace } from "@/features/role-profiles/skills-workspace";
import type { RoleProfile } from "@/lib/role-profile";

afterEach(() => {
  cleanup();
});

function profile(): RoleProfile {
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
      revision: 2,
      status: "profile_draft",
      purpose: "Own regional revenue",
      responsibilities: "Lead the sales cycle",
      businessOutcomes: "Grow revenue",
      decisionScope: "Approve discounts",
      industry: "Business services",
      region: "West Africa",
      teamSize: 8,
      reportingLevel: "manager",
      positionId: "position-1",
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
          criteria: "Review a territory plan",
          provenance: "ai_suggested",
          position: 1,
          createdAt: "2026-10-29T10:00:00Z",
          updatedAt: "2026-10-29T10:00:00Z",
        },
        {
          id: "assignment-2",
          skillId: "skill-2",
          skillName: "Pricing policy",
          category: "domain",
          definition: "Knows the pricing rules",
          requiredProficiency: 2,
          priority: "preferred",
          rationale: "The role approves discounts",
          provenance: "ai_suggested",
          position: 2,
        },
      ],
      ai: {
        model: "mock-role-skills",
        sourceContextVersion: "1",
        generatedAt: "2026-10-29T10:00:00Z",
        draft: {
          skills: [
            { skillId: "skill-1", kind: "technical" },
            { skillId: "skill-2", kind: "knowledge" },
          ],
        },
      },
      createdAt: "2026-10-23T00:00:00Z",
    },
  };
}

describe("skills workspace", () => {
  it("shows mock suggestions by category without treating them as approved", () => {
    render(<SkillsWorkspace profile={profile()} jobModel="mock-role-skills" />);
    expect(screen.getByRole("heading", { name: "Area Sales Manager" })).toBeInTheDocument();
    expect(screen.getByText("Development mock. These suggestions are not model output.")).toBeInTheDocument();
    expect(screen.getByText("These recommendations are suggestions pending human review.")).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Technical Skills (1)" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("tab", { name: "Knowledge (1)" })).toHaveAttribute("aria-selected", "false");
    expect(screen.getByText("Territory planning")).toBeInTheDocument();
    expect(screen.queryByText("Pricing policy")).not.toBeInTheDocument();
    expect(screen.getAllByText("AI suggestion").length).toBeGreaterThan(0);
    expect(screen.getByText("Rationale: The role owns coverage")).toBeInTheDocument();
    expect(screen.queryByText(/^approved$/i)).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Add a skill" })).toHaveAttribute("href", "/admin/role-profiles/profile-1#assign-skill");
    expect(screen.getByRole("link", { name: "Review and customise" })).toHaveAttribute("href", "/admin/role-profiles/profile-1");
  });
});