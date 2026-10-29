import { describe, expect, it } from "vitest";

import { proficiencyPercentage } from "@/features/role-profiles/gap-scale";

describe("proficiency scale", () => {
  it("stores a 0 to 4 level as a percentage the gap service can divide by 25", () => {
    expect(proficiencyPercentage(0)).toBe(0);
    expect(proficiencyPercentage(3)).toBe(75);
    expect(proficiencyPercentage(4)).toBe(100);
    expect(proficiencyPercentage(1.5)).toBeNull();
    expect(proficiencyPercentage(5)).toBeNull();
  });
});
