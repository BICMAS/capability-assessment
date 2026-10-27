import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { EmptyState, StatusMessage } from "@/components/ui/status-message";

describe("StatusMessage", () => {
  it("renders errors with alert role", () => {
    render(<StatusMessage tone="error">Permission denied</StatusMessage>);
    expect(screen.getByRole("alert")).toHaveTextContent("Permission denied");
  });

  it("renders success with status role", () => {
    render(<StatusMessage tone="success">Saved</StatusMessage>);
    expect(screen.getByRole("status")).toHaveTextContent("Saved");
  });

  it("renders nothing when empty", () => {
    const { container } = render(<StatusMessage tone="error">{""}</StatusMessage>);
    expect(container).toBeEmptyDOMElement();
  });
});

describe("EmptyState", () => {
  it("renders empty guidance", () => {
    render(<EmptyState>Nothing here yet.</EmptyState>);
    expect(screen.getByText("Nothing here yet.")).toBeInTheDocument();
  });
});
