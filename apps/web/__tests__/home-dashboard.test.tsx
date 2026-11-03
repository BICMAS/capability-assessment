import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { HomeDashboard } from "@/features/home/dashboard";

afterEach(() => {
  cleanup();
});

describe("home dashboard", () => {
  it("shows live metrics, readable activity, quick actions, and approval counts", () => {
    render(
      <HomeDashboard
        userName="Ada Admin"
        metrics={[
          {
            label: "Frameworks",
            href: "/admin?focus=frameworks",
            action: "Open editor",
            value: "3",
            detail: "Total available frameworks",
            icon: "frameworks",
          },
          {
            label: "Assignments",
            href: "/admin?focus=assignments",
            action: "Assign people",
            value: "2",
            detail: "Published versions ready to assign",
            icon: "assignments",
          },
          {
            label: "Plans",
            href: "/admin?focus=plans",
            action: "Create a plan",
            value: "0",
            detail: "Active assessment plans across the organisation",
            icon: "plans",
          },
          {
            label: "Organisation",
            href: "/admin?focus=organisation",
            action: "View organisation",
            value: "1",
            detail: "1 departments recorded",
            icon: "organisation",
          },
        ]}
        records={{
          ok: true,
          value: [
            {
              actorPersonId: "ada",
              organisationId: "bicmas",
              action: "person.signed_in",
              entityType: "person",
              entityId: "ada",
              occurredAt: "2026-10-29T20:52:00Z",
            },
          ],
        }}
        actorNames={new Map([["ada", "Ada Admin"]])}
        showAllRecords={false}
        totalRecords={1}
        approvals={{
          awaiting: { ok: true, value: 2 },
          submitted: { ok: true, value: 0 },
          reviewed: { ok: true, value: 1 },
        }}
      />,
    );

    expect(screen.getByText("Welcome back, Ada Admin. Here's what's happening across your organisation.")).toBeTruthy();
    expect(screen.getByRole("link", { name: /Frameworks/ }).getAttribute("href")).toBe("/admin?focus=frameworks");
    expect(screen.getByText("Ada Admin signed in")).toBeTruthy();
    expect(screen.queryByText("person.signed_in")).toBeNull();
    expect(screen.getByRole("link", { name: /Create a role profile/ }).getAttribute("href")).toBe("/admin/role-profiles/new");
    expect(screen.getByRole("link", { name: /Upload organogram/ }).getAttribute("href")).toBe("/admin/role-profiles?entry=upload");
    expect(screen.getByRole("link", { name: /View approvals/ }).getAttribute("href")).toBe("/admin/role-profiles/review");

    const panel = screen.getByRole("heading", { name: "My approvals" }).closest("section");
    expect(panel).toBeTruthy();
    expect(within(panel as HTMLElement).getByText("2")).toBeTruthy();
    expect(within(panel as HTMLElement).getByRole("link", { name: /Awaiting my review/ }).getAttribute("href")).toBe(
      "/admin/role-profiles/review",
    );
    expect(within(panel as HTMLElement).getByRole("link", { name: /Submitted by me/ }).getAttribute("href")).toBe(
      "/admin/role-profiles/review?view=submitted",
    );
    expect(within(panel as HTMLElement).getByRole("link", { name: /Previously reviewed/ }).getAttribute("href")).toBe(
      "/admin/role-profiles/review?view=reviewed",
    );
  });

  it("shows empty and error states without inventing metrics", () => {
    render(
      <HomeDashboard
        userName="Ada Admin"
        metrics={[
          {
            label: "Frameworks",
            href: "/admin?focus=frameworks",
            action: "Open editor",
            value: "—",
            detail: "Frameworks is unavailable right now.",
            icon: "frameworks",
            tone: "error",
          },
          {
            label: "Assignments",
            href: "/admin?focus=assignments",
            action: "Assign people",
            value: "—",
            detail: "Frameworks is unavailable right now.",
            icon: "assignments",
            tone: "error",
          },
          {
            label: "Plans",
            href: "/admin?focus=plans",
            action: "Create a plan",
            value: "—",
            detail: "Plans is unavailable right now.",
            icon: "plans",
            tone: "error",
          },
          {
            label: "Organisation",
            href: "/admin?focus=organisation",
            action: "View organisation",
            value: "—",
            detail: "Role assignments is unavailable right now.",
            icon: "organisation",
            tone: "error",
          },
        ]}
        records={{ ok: false, error: "Audit record is unavailable right now." }}
        actorNames={new Map()}
        showAllRecords={false}
        totalRecords={0}
        approvals={{
          awaiting: { ok: false, error: "Approvals awaiting review is unavailable right now." },
          submitted: { ok: true, value: 0 },
          reviewed: { ok: true, value: 0 },
        }}
      />,
    );

    expect(screen.getByText("Audit record is unavailable right now.")).toBeTruthy();
    expect(screen.getByText("Approvals awaiting review is unavailable right now.")).toBeTruthy();
    expect(screen.getByText("You have not submitted a profile for review.")).toBeTruthy();
    expect(screen.getByText("No reviewed profiles yet.")).toBeTruthy();
    expect(screen.getAllByText("—").length).toBeGreaterThan(0);
  });
});
