import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("react-dom", async () => {
  const actual = await vi.importActual<typeof import("react-dom")>("react-dom");
  return {
    ...actual,
    useFormStatus: () => ({ pending: false, data: null, method: null, action: null }),
  };
});

import { PendingButton } from "@/components/ui/pending-button";

afterEach(() => {
  cleanup();
});

describe("PendingButton", () => {
  it("renders the idle label", () => {
    render(
      <form>
        <PendingButton>Save</PendingButton>
      </form>,
    );
    expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
  });

  it("honours the disabled prop", () => {
    render(
      <form>
        <PendingButton disabled>Confirm</PendingButton>
      </form>,
    );
    expect(screen.getByRole("button", { name: "Confirm" })).toBeDisabled();
  });
});
