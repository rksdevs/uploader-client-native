/** Mirrors wow-logs.co.in leaderboard styling (percentile tiers, class colours). */

export const toSlug = (str: string | undefined): string => (str ?? "").replace(/\s+/g, "");

export function getPercentileColor(percentile: number): string {
  if (percentile >= 100) return "#e5cc80";
  if (percentile >= 99) return "#e268a8";
  if (percentile >= 95) return "#ff8000";
  if (percentile >= 75) return "#a335ee";
  if (percentile >= 50) return "#0070ff";
  if (percentile >= 30) return "#1eff00";
  return "#808080";
}

const CLASS_COLORS_LIGHT: Record<string, string> = {
  "Death Knight": "#C41E3A",
  "Demon Hunter": "#A330C9",
  Druid: "#FF7C0A",
  Evoker: "#33937F",
  Hunter: "#AAD372",
  Mage: "#3FC7EB",
  Monk: "#00FF98",
  Paladin: "#F48CBA",
  Priest: "#5A5A5A",
  Rogue: "#C79C00",
  Shaman: "#0070DD",
  Warlock: "#8788EE",
  Warrior: "#C69B6D",
  Unknown: "#6B7280",
};

const CLASS_COLORS_DARK: Record<string, string> = {
  ...CLASS_COLORS_LIGHT,
  Priest: "#FFFFFF",
  Rogue: "#FFF468",
};

export function getClassColor(playerClass: string, theme: "light" | "dark"): string {
  const map = theme === "dark" ? CLASS_COLORS_DARK : CLASS_COLORS_LIGHT;
  return map[playerClass] ?? map.Unknown ?? "#6B7280";
}

/** Public URL for Vite `public/` icons. */
export function specPortraitPath(playerClass: string, playerSpec: string): string {
  const cls = toSlug(playerClass);
  const spec = (playerSpec ?? "").trim();
  const specSlug = toSlug(spec);
  if (!spec || spec === "Unknown" || specSlug === "") {
    return `/icons/classes/${cls || "Unknown"}.jpg`;
  }
  const file = `${specSlug.toLowerCase()}${cls.toLowerCase()}.png`;
  return `/icons/specs/${file}`;
}
