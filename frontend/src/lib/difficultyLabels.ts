/**
 * Matches `client/lib/difficulty.ts` — display for leaderboard difficulty enums.
 */

export function formatDifficultyCompact(difficulty: string): string {
  switch (difficulty) {
    case "TEN_NM":
      return "10 NM";
    case "TEN_HC":
      return "10 HC";
    case "TWENTY_FIVE_NM":
      return "25 NM";
    case "TWENTY_FIVE_HC":
      return "25 HC";
    case "OTHERS":
      return "Other";
    default:
      return difficulty;
  }
}

export function isHeroicDifficulty(difficulty: string): boolean {
  return difficulty.includes("HC");
}

/** Skull for heroic, shield for normal (same as website). */
export function difficultyIcon(difficulty: string): string {
  return isHeroicDifficulty(difficulty) ? "💀" : "🛡️";
}

export function difficultyOptionLabel(difficulty: string): string {
  return `${difficultyIcon(difficulty)} ${formatDifficultyCompact(difficulty)}`;
}
