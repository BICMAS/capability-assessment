import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("react-dom", async () => {
  const actual = await vi.importActual<typeof import("react-dom")>("react-dom");
  return { ...actual, useFormStatus: () => ({ pending: false, data: null, method: null, action: null }) };
});

vi.mock("@/features/role-profiles/actions", () => ({
  openRoleProfileForm: vi.fn(),
}));

import { RoleProfileFrame } from "@/features/role-profiles/frame";
import { Organogram } from "@/features/role-profiles/organogram";

afterEach(() => {
  cleanup();
});

describe("role profile directory", () => {
  it("groups people by department and keeps the profile actions", () => {
    render(
      <Organogram
        departmentId=""
        departments={{ ok: true, value: [{ id: "sales", organisationId: "org", name: "Sales" }] }}
        people={{
          ok: true,
          value: [
            { id: "eli", organisationId: "org", name: "Eli Employee", departmentName: "Sales" },
            { id: "ada", organisationId: "org", name: "Ada Admin", departmentName: "" },
          ],
        }}
        roles={{
          ok: true,
          value: [
            {
              id: "role-1",
              organisationId: "org",
              personId: "eli",
              personName: "Eli Employee",
              title: "Area Sales Manager",
              createdAt: "2026-10-27T00:00:00Z",
            },
          ],
        }}
        error=""
      />,
    );

    expect(screen.getByRole("heading", { name: "Sales" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "No department" })).toBeTruthy();
    expect(screen.getByText("Area Sales Manager")).toBeTruthy();
    expect(screen.getByText("No role title")).toBeTruthy();
    expect(screen.getByRole("link", { name: "Create a role profile" }).getAttribute("href")).toBe("/admin/role-profiles/new");
    expect(screen.getByRole("link", { name: "Review inbox" }).getAttribute("href")).toBe("/admin/role-profiles/review");
    expect(screen.queryByRole("navigation", { name: "Role profile workflow" })).toBeNull();
  });

  it("shows the seven-stage workflow before a profile exists", () => {
    const { rerender } = render(
      <RoleProfileFrame current="organogram">
        <p>Directory</p>
      </RoleProfileFrame>,
    );
    expect(screen.getByRole("navigation", { name: "Role skills workflow" })).toBeTruthy();
    expect(screen.getByRole("link", { name: /Upload Organogram/ }).getAttribute("aria-current")).toBe("step");
    expect(screen.queryByRole("link", { name: /Provide Role Information/ })).toBeNull();
    expect(screen.queryByRole("link", { name: /Use the Insights/ })).toBeNull();

    rerender(
      <RoleProfileFrame
        current="skills"
        profile={{
          id: "profile-1",
          organisationId: "org",
          roleId: "role-1",
          roleName: "Area Sales Manager",
          createdByPersonId: "ada",
          createdAt: "2026-10-27T00:00:00Z",
          version: {
            id: "version-1",
            profileId: "profile-1",
            organisationId: "org",
            versionNumber: 1,
            revision: 1,
            status: "profile_draft",
            purpose: "",
            responsibilities: "",
            businessOutcomes: "",
            decisionScope: "",
            industry: "",
            region: "",
            teamSize: null,
            reportingLevel: "",
            frameworkVersionId: "",
            sources: [],
            skills: [],
            ai: { model: "", sourceContextVersion: "", generatedAt: null, draft: null },
            createdAt: "2026-10-27T00:00:00Z",
          },
        }}
      >
        <p>Skills</p>
      </RoleProfileFrame>,
    );

    expect(screen.getByRole("navigation", { name: "Role skills workflow" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Area Sales Manager" })).toBeTruthy();
    expect(screen.getByRole("link", { name: /Provide Role Information/ }).getAttribute("href")).toBe("/admin/role-profiles/profile-1/context");
    expect(screen.getByRole("link", { name: /Review and Customise Skills/ }).getAttribute("aria-current")).toBe("step");
    expect(screen.queryByRole("link", { name: /Use the Insights/ })).toBeNull();
  });
});
