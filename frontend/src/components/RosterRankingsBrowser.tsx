import React, { useCallback, useEffect, useMemo, useState } from "react";
import { Users } from "lucide-react";
import { toast } from "sonner";
import {
  CommitRaidPartyRankingsJSON,
  FetchLeaderboardFilterRaidsJSON,
  FetchLeaderboardSeasonsConfigJSON,
  FetchRosterCharacterRankingsJSON,
} from "../../wailsjs/go/main/App";
import { difficultyOptionLabel } from "../lib/difficultyLabels";
import {
  getClassColor,
  getPercentileColor,
  specPortraitPath,
  toSlug,
} from "../lib/leaderboardVisuals";

// ─── types ────────────────────────────────────────────────────────────────────

type FilterNum = { label: string; value: number };

type RosterExport = {
  members?: { name: string; realm?: string; class?: string }[];
  memberCount?: number;
  groupType?: string;
};

type SpecPointEntry = {
  spec: string;
  points: number;
};

type CharacterRankingRow = {
  playerName: string;
  class: string;
  avgPercentile: number;
  allStarPoints: number;
  specPoints?: SpecPointEntry[];
  bossPercentiles: Record<string, number | null>;
};

type RosterAPIResponse = {
  rankings: CharacterRankingRow[];
  bossOrder: string[];
  meta: {
    serverId: number;
    raidId: number;
    difficulty: string;
    season: number;
    ladder: string;
    groupType?: string;
    requestedCount: number;
    matchedCount: number;
    notFound?: string[];
  };
};

// ─── constants ────────────────────────────────────────────────────────────────

const DIFFICULTIES: { value: string; label: string }[] = [
  { value: "TEN_NM", label: difficultyOptionLabel("TEN_NM") },
  { value: "TEN_HC", label: difficultyOptionLabel("TEN_HC") },
  { value: "TWENTY_FIVE_NM", label: difficultyOptionLabel("TWENTY_FIVE_NM") },
  { value: "TWENTY_FIVE_HC", label: difficultyOptionLabel("TWENTY_FIVE_HC") },
  { value: "OVERALL", label: "Overall (weighted)" },
];

const LADDERS = [
  { value: "HARDCORE", label: "Hardcore" },
  { value: "COMPETITIVE", label: "Competitive" },
  { value: "REGULAR", label: "Regular" },
];

// ─── helpers ──────────────────────────────────────────────────────────────────

function parseFilterNumJSON(json: string): FilterNum[] {
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

/** Hand-crafted short labels for every known WotLK / Cata boss. Falls back to 8-char trim. */
const BOSS_SHORT_LABEL: Record<string, string> = {
  // ── Naxxramas ──────────────────────────────────────────────
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
  // ── Ulduar ────────────────────────────────────────────────
  "Flame Leviathan": "Fl. Lev",
  "Ignis the Furnace Master": "Ignis",
  Razorscale: "Razorsc",
  "XT-002 Deconstructor": "XT-002",
  "Assembly of Iron": "Assemb",
  Kologarn: "Kologarn",
  Auriaya: "Auriaya",
  Hodir: "Hodir",
  Thorim: "Thorim",
  Freya: "Freya",
  Mimiron: "Mimiron",
  "General Vezax": "Vezax",
  "Yogg-Saron": "Yogg-S",
  "Algalon the Observer": "Algalon",
  // ── Trial of the Crusader ────────────────────────────────
  "Northrend Beasts": "Beasts",
  "Lord Jaraxxus": "Jaraxxus",
  "Faction Champions": "Faction",
  "Twin Val'kyr": "Twin V",
  "Anub'arak": "Anub'ark",
  // ── Icecrown Citadel ─────────────────────────────────────
  "Lord Marrowgar": "Marrow",
  "Lady Deathwhisper": "Lady D",
  "Gunship Battle": "Gunship",
  "Deathbringer Saurfang": "DBS",
  Festergut: "Fester",
  Rotface: "Rotface",
  "Professor Putricide": "PP",
  "Blood Prince Council": "BPC",
  "Blood-Queen Lana'thel": "BQL",
  "Valithria Dreamwalker": "Valithr",
  Sindragosa: "Sindra",
  "The Lich King": "LK",
  // ── Other WotLK / Ruby Sanctum ───────────────────────────
  "General Zarithrian":      "Zarith",
  "Saviana Ragefire":        "Saviana",
  "Baltharus the Warborn":   "Balthar",
  Malygos: "Malygos",
  Sartharion: "Sarthar",
  Onyxia: "Onyxia",
  Halion: "Halion",
  "Archavon the Stone Watcher": "Archavon",
  "Emalon the Storm Watcher": "Emalon",
  "Koralon the Flame Watcher": "Koralon",
  "Toravon the Ice Watcher": "Toravon",
  // ── Cataclysm ────────────────────────────────────────────
  Magmaw: "Magmaw",
  "Omnotron Defense System": "Omnotrn",
  Maloriak: "Maloriak",
  Atramedes: "Atram",
  Chimaeron: "Chimae",
  Nefarian: "Nefarian",
  "Halfus Wyrmbreaker": "Halfus",
  "Valiona and Theralion": "Val&Ther",
  "Ascendant Council": "Ascend",
  "Cho'gall": "Cho'gall",
  Sinestra: "Sinestra",
  "Conclave of Wind": "Conclav",
  "Al'Akir": "Al'Akir",
  Shannox: "Shannox",
  "Lord Rhyolith": "Rhyolth",
  "Beth'tilac": "Beth'tl",
  "Baleroc, the Gatekeeper": "Baleroc",
  Alysrazor: "Alysraz",
  "Majordomo Staghelm": "Mj.Stgh",
  Ragnaros: "Ragnaros",
  Argaloth: "Argaloth",
  "Occu'thar": "Occu'th",
  Alizabal: "Alizabal",
  Morchok: "Morchok",
  "Warlord Zon'ozz": "Zon'ozz",
  "Yor'sahj the Unsleeping": "Yor'sahj",
  "Hagara the Stormbinder": "Hagara",
  Ultraxion: "Ultraxi",
  "Warmaster Blackhorn": "Blkhorn",
  "Spine of Deathwing": "Spine",
  "Madness of Deathwing": "Madness",
};

function shortBoss(name: string): string {
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
  const primary = playerSpec
    ? specPortraitPath(playerClass, playerSpec)
    : classJpg;
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

// ─── component ────────────────────────────────────────────────────────────────

interface Props {
  serverNumericId: number | null;
  wowDirectory: string;
  disabled: boolean;
  theme?: "light" | "dark";
}

export default function RosterRankingsBrowser({
  serverNumericId,
  wowDirectory,
  disabled,
  theme = "dark",
}: Props) {
  const [rosterPaste, setRosterPaste] = useState("");
  const [season, setSeason] = useState(4);
  const [seasonMode, setSeasonMode] = useState<"active" | "specific">("active");
  const [raidId, setRaidId] = useState<number | null>(null);
  const [raidName, setRaidName] = useState("");
  const [difficulty, setDifficulty] = useState("TWENTY_FIVE_HC");
  const [ladder, setLadder] = useState("REGULAR");
  const [raids, setRaids] = useState<FilterNum[]>([]);
  const [loading, setLoading] = useState(false);
  const [committing, setCommitting] = useState(false);
  const [apiData, setApiData] = useState<RosterAPIResponse | null>(null);

  const serverOk = serverNumericId != null && serverNumericId > 0;

  // Load active season from config
  useEffect(() => {
    FetchLeaderboardSeasonsConfigJSON()
      .then((json: string) => {
        const cfg = JSON.parse(json) as { active?: Record<string, number> };
        const active = cfg.active ?? {};
        const first = Object.values(active)[0];
        if (typeof first === "number" && first > 0) setSeason(first);
      })
      .catch(() => {});
  }, []);

  // Load raids when server changes
  useEffect(() => {
    if (!serverOk) {
      setRaids([]);
      return;
    }
    FetchLeaderboardFilterRaidsJSON(
      serverNumericId!,
      seasonMode === "specific" ? season : 0,
    )
      .then((json: string) => {
        const list = parseFilterNumJSON(json);
        setRaids(list);
        if (list.length > 0 && raidId == null) {
          setRaidId(list[0].value);
          setRaidName(list[0].label);
        }
      })
      .catch(() => setRaids([]));
  }, [serverNumericId, season, seasonMode, serverOk]);

  const rosterPreview = useMemo(() => {
    if (!rosterPaste.trim()) return null;
    try {
      const r = JSON.parse(rosterPaste.trim()) as RosterExport;
      const n = r.members?.length ?? r.memberCount ?? 0;
      return { ok: true as const, count: n, groupType: r.groupType ?? "?" };
    } catch {
      return { ok: false as const };
    }
  }, [rosterPaste]);

  const buildFiltersJSON = useCallback(() => {
    if (!serverOk || raidId == null) throw new Error("Select server and raid.");
    return JSON.stringify({
      serverId: serverNumericId,
      raidId,
      difficulty,
      season,
      ladder: ladder.toUpperCase(),
      raidName,
    });
  }, [serverOk, serverNumericId, raidId, difficulty, season, ladder, raidName]);

  const handleFetch = () => {
    if (!rosterPaste.trim()) {
      toast.error("Paste raid/party JSON from the WoW addon first.");
      return;
    }
    if (!serverOk || raidId == null) {
      toast.error("Select server and raid.");
      return;
    }
    setLoading(true);
    setApiData(null);
    let filtersJson: string;
    try {
      filtersJson = buildFiltersJSON();
    } catch (e) {
      toast.error(String(e));
      setLoading(false);
      return;
    }

    FetchRosterCharacterRankingsJSON(rosterPaste.trim(), filtersJson)
      .then((json: string) => {
        const data = JSON.parse(json) as RosterAPIResponse;
        setApiData(data);
        const m = data.meta;
        toast.success(
          `Loaded ${data.rankings?.length ?? 0} players (${m.matchedCount}/${m.requestedCount} matched)`,
        );
        if (m.notFound?.length) {
          toast.message(`${m.notFound.length} not on site`, {
            description: m.notFound.slice(0, 8).join(", "),
          });
        }
      })
      .catch((err: unknown) =>
        toast.error("Failed to fetch roster rankings", {
          description: String(err),
        }),
      )
      .finally(() => setLoading(false));
  };

  const handleSend = () => {
    if (!wowDirectory.trim()) {
      toast.error(
        "Link your WoW directory before writing RankingsPayload.lua.",
      );
      return;
    }
    if (!apiData) {
      toast.error("Fetch rankings first.");
      return;
    }
    setCommitting(true);
    CommitRaidPartyRankingsJSON(JSON.stringify(apiData), raidName)
      .then((msg: string) => toast.success(msg, { duration: 14000 }))
      .catch((err: unknown) =>
        toast.error("Failed to write addon file", { description: String(err) }),
      )
      .finally(() => setCommitting(false));
  };

  const bossOrder = apiData?.bossOrder ?? [];
  const canFetch =
    !disabled &&
    !loading &&
    serverOk &&
    raidId != null &&
    rosterPreview?.ok === true;
  const canSend =
    !disabled && !committing && !!apiData && !!wowDirectory.trim();

  return (
    <section
      className="redesign-card rankings-browser rankings-browser--compact"
      style={{ marginTop: 12 }}
    >
      {/* ── header ── */}
      <div className="card-head-row rankings-browser__head">
        <Users
          size={20}
          strokeWidth={2}
          className="card-head-icon"
          aria-hidden
        />
        <h2 className="rankings-browser__title" style={{ flex: 1, margin: 0 }}>
          Raid / Party Rankings
        </h2>
      </div>

      <p className="rankings-browser__lead rankings-browser__lead--tight">
        In WoW: <strong>Export Raid/Party</strong> → copy JSON → paste below →
        fetch → send to addon → <strong>/reload</strong> → open{" "}
        <strong>Raid/Party Rankings</strong> in-game.
      </p>

      {/* ── roster paste ── */}
      <label
        className="rb-field rb-field--full"
        style={{ marginBottom: "0.5rem" }}
      >
        <span className="rb-field__label">
          Roster export (paste from WoW addon)
        </span>
        <textarea
          className="rb-field__control"
          style={{
            resize: "vertical",
            minHeight: 72,
            fontFamily: "ui-monospace, monospace",
            fontSize: "0.72rem",
          }}
          placeholder='{"groupType":"raid","members":[{"name":"Ahalpuh","realm":"ChromieCraft"},...], ...}'
          value={rosterPaste}
          onChange={(e) => {
            setRosterPaste(e.target.value);
            setApiData(null);
          }}
          disabled={disabled}
          rows={3}
        />
      </label>

      {rosterPreview != null && (
        <p className="rb-field__microcopy" style={{ marginBottom: "0.5rem" }}>
          {rosterPreview.ok ? (
            `${rosterPreview.count} members · ${rosterPreview.groupType}`
          ) : (
            <span style={{ color: "var(--rb-warn, #c9a227)" }}>
              Invalid JSON — paste the full addon export.
            </span>
          )}
        </p>
      )}

      {/* ── filters ── */}
      <div
        className="rankings-browser__grid"
        style={{
          gridTemplateColumns: "repeat(4, minmax(0, 1fr))",
          marginBottom: "0.55rem",
        }}
      >
        <div className="rb-field rb-field--season">
          <span className="rb-field__label">Season</span>
          <select
            className="form-select rb-field__control"
            value={seasonMode}
            onChange={(e) =>
              setSeasonMode(e.target.value as "active" | "specific")
            }
            disabled={disabled || loading}
          >
            <option value="active">Active</option>
            <option value="specific">Specific…</option>
          </select>
          {seasonMode === "specific" && (
            <select
              className="form-select rb-field__control rb-field__control--stacked"
              value={season}
              onChange={(e) => setSeason(parseInt(e.target.value, 10) || 1)}
              disabled={disabled || loading}
            >
              {[1, 2, 3, 4, 5].map((s) => (
                <option key={s} value={s}>
                  Season {s}
                </option>
              ))}
            </select>
          )}
        </div>

        <label className="rb-field">
          <span className="rb-field__label">Raid</span>
          <select
            className="form-select rb-field__control"
            value={raidId ?? ""}
            onChange={(e) => {
              const id = parseInt(e.target.value, 10);
              setRaidId(id);
              setRaidName(raids.find((r) => r.value === id)?.label ?? "");
              setApiData(null);
            }}
            disabled={disabled || loading || !serverOk || raids.length === 0}
          >
            {raids.length === 0 && <option value="">Loading…</option>}
            {raids.map((r) => (
              <option key={r.value} value={r.value}>
                {r.label}
              </option>
            ))}
          </select>
        </label>

        <label className="rb-field">
          <span className="rb-field__label">Difficulty</span>
          <select
            className="form-select rb-field__control"
            value={difficulty}
            onChange={(e) => {
              setDifficulty(e.target.value);
              setApiData(null);
            }}
            disabled={disabled || loading}
          >
            {DIFFICULTIES.map((d) => (
              <option key={d.value} value={d.value}>
                {d.label}
              </option>
            ))}
          </select>
        </label>

        <label className="rb-field">
          <span className="rb-field__label">Ladder</span>
          <select
            className="form-select rb-field__control"
            value={ladder}
            onChange={(e) => {
              setLadder(e.target.value);
              setApiData(null);
            }}
            disabled={disabled || loading}
          >
            {LADDERS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        </label>
      </div>

      {/* ── actions ── */}
      <div
        className="rankings-browser__actions"
        style={{ marginBottom: "0.45rem" }}
      >
        <button
          type="button"
          className="btn-gradient-primary"
          onClick={handleFetch}
          disabled={!canFetch}
        >
          {loading ? "Fetching…" : "Get raid/party rankings"}
        </button>
        <button
          type="button"
          className="btn-send-addon"
          onClick={handleSend}
          disabled={!canSend}
        >
          {committing ? "Writing…" : "Send to addon"}
        </button>
      </div>

      {/* ── results meta ── */}
      {apiData?.meta && (
        <p className="rb-field__microcopy" style={{ marginBottom: "0.4rem" }}>
          {apiData.meta.matchedCount}/{apiData.meta.requestedCount} members
          matched
          {apiData.meta.notFound?.length
            ? ` · ${apiData.meta.notFound.length} not on site: ${apiData.meta.notFound.slice(0, 4).join(", ")}${apiData.meta.notFound.length > 4 ? "…" : ""}`
            : ""}
        </p>
      )}

      {/* ── table ── */}
      {apiData && apiData.rankings.length > 0 && (
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
                <th
                  className="rr-col-num"
                  title="Average percentile across all bosses"
                >
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
              {apiData.rankings.map((row, idx) => {
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
                        <div style={{ display: "flex", flexDirection: "column", gap: 2, alignItems: "flex-end" }}>
                          {row.specPoints.map((sp, idx) => {
                            const labelColors = ["#c084fc", "#60a5fa", "#34d399"];
                            const labelColor = labelColors[idx] ?? labelColors[labelColors.length - 1];
                            const shortName = sp.spec.slice(0, 4);
                            return (
                              <span
                                key={sp.spec}
                                style={{ lineHeight: 1.2, whiteSpace: "nowrap", color: "rgba(255,255,255,0.9)" }}
                              >
                                <span style={{ color: labelColor, marginRight: 3, fontWeight: 600 }}>{shortName}</span>
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
      )}

      {apiData && apiData.rankings.length === 0 && (
        <p className="rankings-browser__note">
          No ranking data found for this roster and filters.
        </p>
      )}
    </section>
  );
}
