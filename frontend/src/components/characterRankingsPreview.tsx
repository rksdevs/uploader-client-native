import React from "react";
import { difficultyOptionLabel } from "../lib/difficultyLabels";
import {
  getClassColor,
  getPercentileColor,
  specPortraitPath,
  toSlug,
} from "../lib/leaderboardVisuals";

export type SpecPointEntry = {
  spec: string;
  points: number;
};

export type CharacterRankingRow = {
  playerName: string;
  class: string;
  avgPercentile: number;
  allStarPoints: number;
  specPoints?: SpecPointEntry[];
  bossPercentiles: Record<string, number | null>;
};

/** Normalize API / Go JSON rows for the preview table (handles numeric strings). */
export function normalizeCharacterRankingRows(
  rows: unknown[],
): CharacterRankingRow[] {
  return rows.map((raw) => {
    const r = raw as Record<string, unknown>;
    const bossPercentiles: Record<string, number | null> = {};
    const bp = r.bossPercentiles as Record<string, unknown> | undefined;
    if (bp && typeof bp === "object") {
      for (const [name, val] of Object.entries(bp)) {
        if (val == null || val === "") {
          bossPercentiles[name] = null;
        } else {
          const n = Number(val);
          bossPercentiles[name] = Number.isFinite(n) ? n : null;
        }
      }
    }
    let specPoints: SpecPointEntry[] | undefined;
    if (Array.isArray(r.specPoints)) {
      specPoints = r.specPoints
        .map((sp) => {
          const o = sp as Record<string, unknown>;
          return {
            spec: String(o.spec ?? ""),
            points: Number(o.points ?? 0),
          };
        })
        .filter((sp) => sp.spec);
    }
    return {
      playerName: String(r.playerName ?? ""),
      class: String(r.class ?? ""),
      avgPercentile: Number(r.avgPercentile ?? 0),
      allStarPoints: Number(r.allStarPoints ?? 0),
      specPoints,
      bossPercentiles,
    };
  });
}

export const DIFFICULTIES: { value: string; label: string }[] = [
  { value: "TEN_NM", label: difficultyOptionLabel("TEN_NM") },
  { value: "TEN_HC", label: difficultyOptionLabel("TEN_HC") },
  { value: "TWENTY_FIVE_NM", label: difficultyOptionLabel("TWENTY_FIVE_NM") },
  { value: "TWENTY_FIVE_HC", label: difficultyOptionLabel("TWENTY_FIVE_HC") },
  { value: "OVERALL", label: "Overall (weighted)" },
];

export const LADDERS = [
  { value: "HARDCORE", label: "Hardcore" },
  { value: "COMPETITIVE", label: "Competitive" },
  { value: "REGULAR", label: "Regular" },
];

export type FilterNum = { label: string; value: number };
export type FilterStr = { label: string; value: string };

export function parseFilterStrJSON(json: string): FilterStr[] {
  try {
    const data = JSON.parse(json) as unknown;
    if (!Array.isArray(data)) return [];
    return data
      .map((row) => {
        const o = row as Record<string, unknown>;
        const label = String(o.label ?? o.Label ?? "").trim();
        const value = String(o.value ?? o.Value ?? "").trim();
        if (!label || !value) return null;
        return { label, value };
      })
      .filter((x): x is FilterStr => x != null);
  } catch {
    return [];
  }
}

export function parseFilterNumJSON(json: string): FilterNum[] {
  try {
    const data = JSON.parse(json) as unknown;
    if (!Array.isArray(data)) return [];
    return data
      .map((row) => {
        const o = row as Record<string, unknown>;
        const label = String(o.label ?? o.Label ?? "").trim();
        const val = o.value ?? o.Value;
        const n = typeof val === "number" ? val : parseInt(String(val), 10);
        if (!label || !Number.isFinite(n) || n <= 0) return null;
        return { label, value: n };
      })
      .filter((x): x is FilterNum => x != null);
  } catch {
    return [];
  }
}

const BOSS_SHORT_LABEL: Record<string, string> = {
  "Anub'Rekhan": "Anub'R",
  "Grand Widow Faerlina": "GW Faerl",
  Maexxna: "Maexxna",
  "Noth the Plaguebringer": "Noth",
  "Heigan the Unclean": "Heigan",
  Loatheb: "Loatheb",
  "Instructor Razuvious": "Ins-Raz",
  "Gothik the Harvester": "Gothik",
  "The Four Horsemen": "4 Horse",
  Patchwerk: "Patchw",
  Grobbulus: "Grobb",
  Gluth: "Gluth",
  Thaddius: "Thaddius",
  Sapphiron: "Sapphi",
  "Kel'Thuzad": "Kel'Thz",
};

export function shortBoss(name: string): string {
  if (!name) return "?";
  const trimmed = name.trim();
  if (BOSS_SHORT_LABEL[trimmed]) return BOSS_SHORT_LABEL[trimmed];
  if (trimmed.length <= 8) return trimmed;
  return trimmed.slice(0, 7) + "…";
}

function SpecPortrait({
  playerClass,
  playerSpec,
}: {
  playerClass: string;
  playerSpec?: string;
}) {
  const classJpg = `/icons/classes/${toSlug(playerClass) || "Unknown"}.jpg`;
  const primary = playerSpec ? specPortraitPath(playerClass, playerSpec) : classJpg;
  const [src, setSrc] = React.useState(primary);
  React.useEffect(() => {
    setSrc(primary);
  }, [primary]);
  return (
    <img
      src={src}
      alt=""
      width={16}
      height={16}
      className="rankings-browser__spec-icon"
      loading="lazy"
      decoding="async"
      onError={() => setSrc((prev) => (prev !== classJpg ? classJpg : prev))}
    />
  );
}

function PercentileCell({ value }: { value: number | null }) {
  if (value == null)
    return <span style={{ color: "var(--text-muted)", opacity: 0.5 }}>—</span>;
  const color = getPercentileColor(value);
  return (
    <span
      style={{
        color,
        fontWeight: 700,
        fontFamily: "ui-monospace, monospace",
        fontSize: "0.7rem",
      }}
    >
      {value.toFixed(0)}
    </span>
  );
}

type TableProps = {
  rankings: CharacterRankingRow[];
  bossOrder: string[];
  theme?: "light" | "dark";
};

export function CharacterRankingsTable({
  rankings,
  bossOrder,
  theme = "dark",
}: TableProps) {
  if (!rankings.length) return null;
  return (
    <div
      className="rankings-browser__table-wrap"
      style={{ overflowX: "auto" }}
    >
      <table
        className="rankings-browser__perf-table rr-boss-table"
        style={{ minWidth: `${340 + bossOrder.length * 46}px` }}
      >
        <thead>
          <tr>
            <th className="rb-col-rank">#</th>
            <th className="rb-col-player">Player</th>
            <th className="rr-col-num" title="Average percentile across all bosses">
              Avg %
            </th>
            <th className="rr-col-num" title="Total Boss Points">
              Points
            </th>
            {bossOrder.map((b) => (
              <th key={b} className="rr-col-boss" title={b}>
                {shortBoss(b)}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rankings.map((row, idx) => {
            const nameColor = getClassColor(row.class, theme);
            return (
              <tr key={`${row.playerName}-${idx}`}>
                <td className="rb-col-rank">#{idx + 1}</td>
                <td className="rb-col-player">
                  <div className="rankings-browser__player-cell">
                    <SpecPortrait playerClass={row.class} />
                    <span
                      className="rankings-browser__player-name"
                      style={{ color: nameColor }}
                    >
                      {row.playerName}
                    </span>
                  </div>
                </td>
                <td className="rr-col-num">
                  <PercentileCell value={row.avgPercentile} />
                </td>
                <td
                  className="rr-col-num"
                  style={{
                    fontVariantNumeric: "tabular-nums",
                    fontFamily: "ui-monospace, monospace",
                    fontSize: "0.7rem",
                    verticalAlign: "middle",
                    padding: "2px 4px",
                  }}
                >
                  {row.specPoints && row.specPoints.length > 1 ? (
                    <div
                      style={{
                        display: "flex",
                        flexDirection: "column",
                        gap: 2,
                        alignItems: "flex-end",
                      }}
                    >
                      {row.specPoints.map((sp, spIdx) => {
                        const labelColors = ["#c084fc", "#60a5fa", "#34d399"];
                        const labelColor =
                          labelColors[spIdx] ?? labelColors[labelColors.length - 1];
                        const shortName = sp.spec.slice(0, 4);
                        return (
                          <span
                            key={sp.spec}
                            style={{
                              lineHeight: 1.2,
                              whiteSpace: "nowrap",
                              color: "rgba(255,255,255,0.9)",
                            }}
                          >
                            <span
                              style={{
                                color: labelColor,
                                marginRight: 3,
                                fontWeight: 600,
                              }}
                            >
                              {shortName}
                            </span>
                            {sp.points.toFixed(1)}
                          </span>
                        );
                      })}
                    </div>
                  ) : (
                    row.allStarPoints.toFixed(1)
                  )}
                </td>
                {bossOrder.map((b) => {
                  const p = row.bossPercentiles?.[b] ?? null;
                  return (
                    <td key={b} className="rr-col-boss">
                      <PercentileCell value={p} />
                    </td>
                  );
                })}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
