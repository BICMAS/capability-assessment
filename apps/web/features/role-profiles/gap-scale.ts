/** Converts an assessed proficiency on the 0–4 scale into the percentage stored by assessment-gap. */
export function proficiencyPercentage(level: number): number | null {
  if (!Number.isInteger(level) || level < 0 || level > 4) {
    return null;
  }
  return level * 25;
}
