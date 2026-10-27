import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

const pathname = vi.hoisted(() => ({ current: "/admin" }));

vi.mock("next/navigation", () => ({
  usePathname: () => pathname.current,
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
  pathname.current = "/admin";
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

    expect(screen.queryByRole("heading", { name: "My capability" })).toBeNull();
    expect(screen.getByRole("link", { name: "My capability" })).toBeTruthy();
    expect(screen.queryByRole("link", { name: "Review" })).toBeNull();
  });

  it("lets a person's name be the heading on their record", () => {
    pathname.current = "/manager/people/eli";
    render(
      <PortalShell area="manager" session={{ ...session, name: "Maya", roles: ["manager"] }}>
        <h2>Eli Employee</h2>
      </PortalShell>,
    );

    expect(screen.queryByRole("heading", { name: "My team" })).toBeNull();
    expect(screen.getByRole("heading", { name: "Eli Employee" })).toBeTruthy();
    expect(screen.getByRole("link", { name: "My team" })).toBeTruthy();
  });
});
