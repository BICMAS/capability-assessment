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

describe("legacy portal shell", () => {
  it("still renders admin navigation for compatibility", () => {
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
  });
});
