import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("next/navigation", () => ({
  usePathname: () => "/admin",
}));

import { PortalShell } from "@/components/portal-shell";

const session = {
  personId: "ada",
  organisationId: "bicmas",
  roles: ["admin"],
  managerOf: [],
  name: "Ada",
};

afterEach(() => {
  cleanup();
});

describe("portal shell", () => {
  it("shows the admin navigation and keeps sign-out", () => {
    render(
      <PortalShell area="admin" session={session}>
        <p>Today</p>
      </PortalShell>,
    );

    expect(screen.getByRole("link", { name: "Home" }).getAttribute("aria-current")).toBe("page");
    expect(screen.getByRole("link", { name: "Role skills profiles" })).toBeTruthy();
    expect(screen.getByRole("link", { name: "Review" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Home" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Sign out" })).toBeTruthy();
    expect(screen.getByText("Ada")).toBeTruthy();
  });

  it("names the employee home from the area, not the admin path", () => {
    render(
      <PortalShell area="employee" session={{ ...session, name: "Eli", roles: ["employee"] }}>
        <p>Record</p>
      </PortalShell>,
    );

    expect(screen.getByRole("heading", { name: "My capability" })).toBeTruthy();
    expect(screen.getByRole("link", { name: "My capability" })).toBeTruthy();
    expect(screen.queryByRole("link", { name: "Review" })).toBeNull();
  });
});
