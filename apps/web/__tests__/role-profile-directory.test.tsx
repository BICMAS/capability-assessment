import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("react-dom", async () => {
  const actual = await vi.importActual<typeof import("react-dom")>("react-dom");
  return { ...actual, useFormStatus: () => ({ pending: false, data: null, method: null, action: null }) };
});

vi.mock("@/features/role-profiles/actions", () => ({
  openRoleProfileForm: vi.fn(),
}));

import { openRoleProfileForm } from "@/features/role-profiles/actions";
import { RoleProfileFrame } from "@/features/role-profiles/frame";
import { Organogram } from "@/features/role-profiles/organogram";

afterEach(() => {
  cleanup();
  vi.mocked(openRoleProfileForm).mockReset();
});

describe("role profile directory", () => {
  it("groups people by department and keeps the profile actions", async () => {
    const user = userEvent.setup();
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

    expect(screen.getByRole("heading", { name: "Organisation directory" })).toBeTruthy();
    expect(screen.getByText(/Departments and role titles come from the organisation directory/i)).toBeTruthy();
    expect(screen.getByText(/organogram records reporting relationships/i)).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Open a profile" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Sales" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "No department" })).toBeTruthy();
    expect(screen.getByText("Area Sales Manager")).toBeTruthy();
    expect(screen.getByText("No role title")).toBeTruthy();
    expect(screen.getByText("Has role title")).toBeTruthy();
    expect(screen.getByText("Needs role title")).toBeTruthy();
    expect(screen.getByRole("link", { name: "Create profile" }).getAttribute("href")).toBe(
      "/admin/role-profiles/new?title=Area%20Sales%20Manager",
    );
    expect(screen.getByRole("link", { name: "Create without title" }).getAttribute("href")).toBe("/admin/role-profiles/new");
    expect(screen.getByRole("link", { name: "Create a role profile" }).getAttribute("href")).toBe("/admin/role-profiles/new");
    expect(screen.getByRole("link", { name: "Review inbox" }).getAttribute("href")).toBe("/admin/role-profiles/review");
    expect(screen.getByLabelText("Profile id")).toBeTruthy();
    expect(screen.queryByRole("navigation", { name: "Role profile workflow" })).toBeNull();

    await user.type(screen.getByRole("searchbox", { name: "Search the directory" }), "eli");
    expect(screen.getByText("Eli Employee")).toBeTruthy();
    expect(screen.queryByText("Ada Admin")).toBeNull();

    await user.clear(screen.getByRole("searchbox", { name: "Search the directory" }));
    await user.click(screen.getByRole("button", { name: "Without role title" }));
    expect(screen.getByText("Ada Admin")).toBeTruthy();
    expect(screen.queryByText("Eli Employee")).toBeNull();
  });

  it("validates the profile id before opening", async () => {
    const user = userEvent.setup();
    render(
      <Organogram
        departmentId=""
        departments={{ ok: true, value: [] }}
        people={{ ok: true, value: [] }}
        roles={{ ok: true, value: [] }}
        error=""
        openError=""
      />,
    );

    const panel = screen.getByRole("heading", { name: "Open a profile" }).closest("aside");
    expect(panel).toBeTruthy();
    await user.click(within(panel as HTMLElement).getByRole("button", { name: "Open profile" }));
    expect(within(panel as HTMLElement).getByText("Enter a profile id.")).toBeTruthy();
    expect(openRoleProfileForm).not.toHaveBeenCalled();
  });

  it("shows the seven-stage workflow before a profile exists", () => {
    const { rerender } = render(
      <RoleProfileFrame current="organogram">
        <p>Directory</p>
      </RoleProfileFrame>,
    );
    expect(screen.getByText("Create and manage role skills profiles for your organisation.")).toBeTruthy();
    expect(screen.getByRole("navigation", { name: "Role skills workflow" })).toBeTruthy();
    expect(screen.getByRole("link", { name: /Upload Organogram/ }).getAttribute("aria-current")).toBe("step");
    expect(screen.getByText("Upload a spreadsheet, build the chart, or import the template.")).toBeTruthy();
    expect(screen.queryByText("Select the organogram position and record the purpose, outcomes, responsibilities, and context for that role.")).toBeNull();
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
    expect(screen.queryByText("Create and manage role skills profiles for your organisation.")).toBeNull();
    expect(screen.getByRole("link", { name: /Provide Role Information/ }).getAttribute("href")).toBe("/admin/role-profiles/profile-1/context");
    expect(screen.getByRole("link", { name: /Review and Customise Skills/ }).getAttribute("aria-current")).toBe("step");
    expect(screen.queryByRole("link", { name: /Use the Insights/ })).toBeNull();
  });
});
