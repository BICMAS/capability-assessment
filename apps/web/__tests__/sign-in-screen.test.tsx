import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import LoginPage from "@/app/page";

afterEach(() => {
  cleanup();
});

describe("login screen", () => {
  it("shows the split sign-in form and the demo accounts", async () => {
    render(await LoginPage({ searchParams: Promise.resolve({}) }));

    expect(screen.getByRole("heading", { name: "Learn" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Log in" })).toBeTruthy();
    expect(screen.getByText("AI coaches. Managers develop. People perform.")).toBeTruthy();
    expect(screen.getByText("New here? Use a demo account.")).toBeTruthy();
    expect(screen.getByLabelText("Email address")).toBeTruthy();
    expect(screen.getByLabelText("Password")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Log in" })).toBeTruthy();
    expect(screen.queryByText("Sign-in failed.")).toBeNull();
  });

  it("shows the sign-in error from the query string", async () => {
    render(await LoginPage({ searchParams: Promise.resolve({ error: "sign-in" }) }));

    expect(screen.getByText("Sign-in failed.")).toBeTruthy();
  });
});
