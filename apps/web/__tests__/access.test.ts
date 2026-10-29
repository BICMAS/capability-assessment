import { describe, expect, it } from "vitest";

import { allowsPath, homeFor } from "@/lib/session";

describe("portal access", () => {
  it("sends HR and business leaders to the authoring portal", () => {
    expect(homeFor(["hr"])).toBe("/admin");
    expect(homeFor(["leader"])).toBe("/admin");
    expect(allowsPath("/admin/role-profiles", ["hr"])).toBe(true);
    expect(allowsPath("/admin/role-profiles", ["leader"])).toBe(true);
    expect(allowsPath("/admin/role-profiles", ["employee"])).toBe(false);
    expect(allowsPath("/manager", ["manager"])).toBe(true);
    expect(allowsPath("/employee", ["leader"])).toBe(false);
  });
});
