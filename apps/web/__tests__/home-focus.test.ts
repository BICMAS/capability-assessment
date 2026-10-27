import { describe, expect, it } from "vitest";

import { homeFocus } from "@/app/(admin)/admin/home-focus";

describe("admin home focus", () => {
  it("stays on the overview until a section is chosen", () => {
    expect(homeFocus({})).toBe("overview");
    expect(homeFocus({ focus: "organisation" })).toBe("organisation");
  });

  it("opens plans from an assignment or plan link", () => {
    expect(homeFocus({ assignment: "a1" })).toBe("plans");
    expect(homeFocus({ plan: "p1" })).toBe("plans");
    expect(homeFocus({ planError: "missing" })).toBe("plans");
  });

  it("keeps assignment results on the assignments section", () => {
    expect(homeFocus({ assignment: "a1", results: "encoded" })).toBe("assignments");
    expect(homeFocus({ assignmentError: "unavailable" })).toBe("assignments");
  });

  it("returns framework errors to the framework editor", () => {
    expect(homeFocus({ importError: "Missing columns" })).toBe("frameworks");
    expect(homeFocus({ review: "blocked", action: "submit", versionId: "v1" })).toBe("frameworks");
  });
});
