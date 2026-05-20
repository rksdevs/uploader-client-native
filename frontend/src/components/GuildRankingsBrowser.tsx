import React, { useCallback, useEffect, useMemo, useState } from "react";
import { ChevronDown, Shield } from "lucide-react";
import { toast } from "sonner";
import {
  CommitGuildRankingsJSON,
  FetchGuildCharacterRankingsJSON,
  FetchGuildInfoJSON,
  FetchLeaderboardFilterDifficultiesJSON,
  FetchLeaderboardFilterRaidsJSON,
  FetchLeaderboardSeasonsConfigJSON,
} from "../../wailsjs/go/main/App";
import {
  CharacterRankingRow,
  CharacterRankingsTable,
  DIFFICULTIES,
  LADDERS,
  normalizeCharacterRankingRows,
  parseFilterNumJSON,
  parseFilterStrJSON,
  type FilterNum,
} from "./characterRankingsPreview";

/** Match web guild rankings: prefer OVERALL, else highest tier with data. */
const DIFFICULTY_DEFAULT_PRIORITY = [
  "OVERALL",
  "TWENTY_FIVE_HC",
  "TEN_HC",
  "TWENTY_FIVE_NM",
  "TEN_NM",
] as const;

type GuildInfo = {
  id: number;
  name: string;
  serverId: number;
  serverName: string;
};

type GuildAPIResponse = {
  rankings: CharacterRankingRow[];
  bossOrder: string[];
  meta: {
    guildId: number;
    guildName?: string;
    serverId: number;
    raidId: number;
    difficulty: string;
    season: number;
    ladder: string;
  };
};

interface Props {
  wowDirectory: string;
  disabled: boolean;
  theme?: "light" | "dark";
}

type SeasonsConfigResponse = {
  active?: Record<string, number>;
  historical?: Record<string, number[]>;
};

function activeSeasonForServerId(
  active: Record<string, number> | undefined,
  serverId: number,
): number | null {
  if (!active || serverId <= 0) return null;
  const v = active[String(serverId)];
  return typeof v === "number" && v > 0 ? v : null;
}

function seasonChoicesForServer(
  config: SeasonsConfigResponse | null,
  serverId: number,
): number[] {
  if (!config || serverId <= 0) return [];
  const sid = String(serverId);
  const set = new Set<number>();
  const active = config.active?.[sid];
  if (typeof active === "number" && active > 0) set.add(active);
  (config.historical?.[sid] ?? []).forEach((n) => {
    if (n > 0) set.add(n);
  });
  return Array.from(set).sort((a, b) => b - a);
}

function RbSelect({
  label,
  value,
  onChange,
  disabled,
  children,
}: {
  label: string;
  value: string | number;
  onChange: (e: React.ChangeEvent<HTMLSelectElement>) => void;
  disabled?: boolean;
  children: React.ReactNode;
}) {
  return (
    <label className="rb-field">
      <span className="rb-field__label">{label}</span>
      <div className="rb-select-wrap">
        <select
          className="form-select rb-field__control"
          value={value}
          onChange={onChange}
          disabled={disabled}
        >
          {children}
        </select>
        <ChevronDown
          className="rb-select-wrap__chevron"
          size={14}
          strokeWidth={2}
          aria-hidden
        />
      </div>
    </label>
  );
}

const FILTER_GRID_STYLE: React.CSSProperties = {
  gridTemplateColumns: "repeat(4, minmax(0, 1fr))",
  marginBottom: "0.55rem",
};

export default function GuildRankingsBrowser({
  wowDirectory,
  disabled,
  theme = "dark",
}: Props) {
  const [guildIdInput, setGuildIdInput] = useState("");
  const [guildInfo, setGuildInfo] = useState<GuildInfo | null>(null);
  const [loadingGuild, setLoadingGuild] = useState(false);
  const [season, setSeason] = useState(0);
  const [seasonMode, setSeasonMode] = useState<"active" | "specific">("active");
  const [raidId, setRaidId] = useState<number | null>(null);
  const [raidName, setRaidName] = useState("");
  const [difficulty, setDifficulty] = useState("OVERALL");
  const [ladder, setLadder] = useState("REGULAR");
  const [raids, setRaids] = useState<FilterNum[]>([]);
  const [loading, setLoading] = useState(false);
  const [committing, setCommitting] = useState(false);
  const [apiData, setApiData] = useState<GuildAPIResponse | null>(null);
  const [seasonsConfig, setSeasonsConfig] = useState<SeasonsConfigResponse | null>(
    null,
  );

  const guildIdNum = useMemo(() => {
    const n = parseInt(guildIdInput.trim(), 10);
    return Number.isFinite(n) && n > 0 ? n : null;
  }, [guildIdInput]);

  useEffect(() => {
    FetchLeaderboardSeasonsConfigJSON()
      .then((json: string) => {
        try {
          setSeasonsConfig(JSON.parse(json) as SeasonsConfigResponse);
        } catch {
          setSeasonsConfig(null);
        }
      })
      .catch(() => setSeasonsConfig(null));
  }, []);

  const activeSeasonForServer = useMemo(
    () => activeSeasonForServerId(seasonsConfig?.active, guildInfo?.serverId ?? 0),
    [seasonsConfig?.active, guildInfo?.serverId],
  );

  const realmSeasonChoices = useMemo(
    () => seasonChoicesForServer(seasonsConfig, guildInfo?.serverId ?? 0),
    [seasonsConfig, guildInfo?.serverId],
  );

  /** Season for API calls — derived from the loaded guild's realm, not the uploader server picker. */
  const effectiveSeason = useMemo(() => {
    if (!guildInfo) return 0;
    if (seasonMode === "active") {
      return activeSeasonForServer ?? season;
    }
    return season;
  }, [guildInfo, seasonMode, activeSeasonForServer, season]);

  const seasonSelectValue =
    seasonMode === "active" ? "active" : String(season || "");

  const applyRealmActiveSeason = useCallback(
    (info: GuildInfo, cfg: SeasonsConfigResponse | null) => {
      const active = activeSeasonForServerId(cfg?.active, info.serverId);
      setSeasonMode("active");
      if (active != null) {
        setSeason(active);
      } else {
        setSeason(0);
        toast.warning("No active season is configured for this realm.", {
          description: `Server id ${info.serverId}. Pick a historical season below.`,
        });
      }
    },
    [],
  );

  const ensureSeasonsConfig = useCallback(async (): Promise<SeasonsConfigResponse | null> => {
    if (seasonsConfig) return seasonsConfig;
    try {
      const json = await FetchLeaderboardSeasonsConfigJSON();
      const cfg = JSON.parse(json) as SeasonsConfigResponse;
      setSeasonsConfig(cfg);
      return cfg;
    } catch {
      return null;
    }
  }, [seasonsConfig]);

  const loadGuild = useCallback(async () => {
    if (!guildIdNum) {
      toast.error("Enter a valid guild ID (number from the guild profile URL).");
      return;
    }
    setLoadingGuild(true);
    setGuildInfo(null);
    setApiData(null);
    setRaidId(null);
    setRaidName("");
    setRaids([]);
    try {
      const [guildJson, cfg] = await Promise.all([
        FetchGuildInfoJSON(guildIdNum),
        ensureSeasonsConfig(),
      ]);
      const info = JSON.parse(guildJson) as GuildInfo;
      setGuildInfo(info);
      applyRealmActiveSeason(info, cfg);
      const active = activeSeasonForServerId(cfg?.active, info.serverId);
      toast.success(
        active != null
          ? `Loaded ${info.name} · active season ${active} for ${info.serverName}`
          : `Loaded guild: ${info.name}`,
      );
    } catch (err: unknown) {
      toast.error("Could not load guild", { description: String(err) });
    } finally {
      setLoadingGuild(false);
    }
  }, [guildIdNum, ensureSeasonsConfig, applyRealmActiveSeason]);

  useEffect(() => {
    if (!guildInfo || !seasonsConfig || seasonMode !== "active") return;
    const active = activeSeasonForServerId(seasonsConfig.active, guildInfo.serverId);
    if (active != null && active !== season) {
      setSeason(active);
    }
  }, [guildInfo, seasonsConfig, seasonMode, season]);

  useEffect(() => {
    if (!guildInfo?.serverId || effectiveSeason <= 0) {
      setRaids([]);
      return;
    }
    let cancelled = false;
    FetchLeaderboardFilterRaidsJSON(
      guildInfo.serverId,
      seasonMode === "specific" ? effectiveSeason : 0,
    )
      .then((json: string) => {
        if (cancelled) return;
        const list = parseFilterNumJSON(json);
        setRaids(list);
        setRaidId((prev) => {
          if (prev != null && list.some((r) => r.value === prev)) return prev;
          const first = list[0];
          if (first) {
            setRaidName(first.label);
            return first.value;
          }
          setRaidName("");
          return null;
        });
      })
      .catch(() => {
        if (!cancelled) setRaids([]);
      });
    return () => {
      cancelled = true;
    };
  }, [guildInfo?.serverId, effectiveSeason, seasonMode]);

  // Default difficulty for this realm/raid/season (same idea as web guild rankings page).
  useEffect(() => {
    if (!guildInfo?.serverId || !raidId || effectiveSeason <= 0) return;
    let cancelled = false;
    FetchLeaderboardFilterDifficultiesJSON(
      guildInfo.serverId,
      raidId,
      seasonMode === "specific" ? effectiveSeason : 0,
    )
      .then((json: string) => {
        if (cancelled) return;
        const opts = parseFilterStrJSON(json).map((o) => o.value);
        const values = new Set([
          ...opts,
          ...DIFFICULTIES.map((d) => d.value),
        ]);
        const pick =
          DIFFICULTY_DEFAULT_PRIORITY.find((d) => values.has(d)) ?? "OVERALL";
        setDifficulty((prev) => {
          if (prev && values.has(prev)) return prev;
          return pick;
        });
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [guildInfo?.serverId, raidId, effectiveSeason, seasonMode]);

  const buildFiltersJSON = useCallback(() => {
    if (!guildInfo || raidId == null)
      throw new Error("Load guild and select raid.");
    return JSON.stringify({
      serverId: guildInfo.serverId,
      guildName: guildInfo.name,
      raidId,
      raidName,
      difficulty,
      season: effectiveSeason,
      ladder: ladder.toUpperCase(),
    });
  }, [guildInfo, raidId, raidName, difficulty, effectiveSeason, ladder]);

  const handleFetch = () => {
    if (!guildIdNum) {
      toast.error("Enter guild ID first.");
      return;
    }
    if (!guildInfo) {
      toast.error("Load guild info first.");
      return;
    }
    if (effectiveSeason <= 0) {
      toast.error("No season for this realm. Load guild again or pick a historical season.");
      return;
    }
    if (raidId == null) {
      toast.error("Select a raid.");
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
    FetchGuildCharacterRankingsJSON(guildIdNum, filtersJson)
      .then((json: string) => {
        const data = JSON.parse(json) as GuildAPIResponse;
        const rankings = normalizeCharacterRankingRows(
          (data.rankings ?? []) as unknown[],
        );
        setApiData({ ...data, rankings });
        const top = rankings[0];
        toast.success(
          `Loaded ${rankings.length} members · ${data.meta.difficulty} · season ${data.meta.season}${
            top && top.avgPercentile > 0
              ? ` · top ${top.playerName} ${top.avgPercentile.toFixed(0)}%`
              : ""
          }`,
        );
      })
      .catch((err: unknown) =>
        toast.error("Failed to fetch guild rankings", {
          description: String(err),
        }),
      )
      .finally(() => setLoading(false));
  };

  const handleSend = () => {
    if (!wowDirectory.trim()) {
      toast.error("Link your WoW directory before writing RankingsPayload.lua.");
      return;
    }
    if (!apiData) {
      toast.error("Fetch rankings first.");
      return;
    }
    setCommitting(true);
    const gName =
      guildInfo?.name ?? apiData.meta.guildName ?? `Guild ${guildIdNum}`;
    CommitGuildRankingsJSON(JSON.stringify(apiData), gName, raidName)
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
    guildIdNum != null &&
    !!guildInfo &&
    effectiveSeason > 0 &&
    raidId != null;
  const canSend =
    !disabled && !committing && !!apiData && !!wowDirectory.trim();

  return (
    <section
      className="redesign-card rankings-browser rankings-browser--compact"
      style={{ marginTop: 12 }}
    >
      <div className="card-head-row rankings-browser__head">
        <Shield size={20} strokeWidth={2} className="card-head-icon" aria-hidden />
        <h2 className="rankings-browser__title" style={{ flex: 1, margin: 0 }}>
          Guild Rankings
        </h2>
      </div>

      <p className="rankings-browser__lead rankings-browser__lead--tight">
        Same table as the{" "}
        <a
          href="https://wow-logs.co.in/guilds/107/rankings"
          target="_blank"
          rel="noreferrer"
          className="rankings-browser__link"
        >
          guild rankings page
        </a>
        . <strong>Load guild</strong> picks this realm&apos;s active season automatically.
        Then fetch → send to addon → <strong>/reload</strong> → open{" "}
        <strong>Guild Rankings</strong> in-game.
      </p>

      <div className="rankings-browser__guild-hint" role="note">
        <strong>Where to find the Guild ID:</strong> open your guild profile on the
        site — the number in the URL is the guild ID. For example,{" "}
        <span className="rankings-browser__guild-hint-example">
          wow-logs.co.in/guilds/107/profile
        </span>{" "}
        means guild ID <strong>107</strong>.
      </div>

      <div className="rankings-browser__guild-setup">
        <div className="rankings-browser__guild-id-field">
          <span className="rb-field__label">Guild ID</span>
          <div className="rankings-browser__guild-id-input-row">
            <input
              type="text"
              className="form-select rb-field__control"
              placeholder="e.g. 107"
              value={guildIdInput}
              onChange={(e) => setGuildIdInput(e.target.value)}
              disabled={disabled}
              aria-label="Guild ID"
            />
            <button
              type="button"
              className="btn-surface rankings-browser__load-guild-btn"
              onClick={loadGuild}
              disabled={disabled || loadingGuild || !guildIdInput.trim()}
            >
              {loadingGuild ? "…" : "Load guild"}
            </button>
          </div>
        </div>

        {guildInfo ? (
          <div className="rankings-browser__guild-loaded">
            <span className="rb-field__label">Loaded guild</span>
            <p className="rankings-browser__guild-detail">
              <strong>{guildInfo.name}</strong>
              <span className="rankings-browser__guild-detail-sep"> · </span>
              <span>{guildInfo.serverName}</span>
              <span className="rankings-browser__guild-detail-meta">
                {" "}
                (server id {guildInfo.serverId}
                {effectiveSeason > 0
                  ? ` · season ${effectiveSeason}${
                      seasonMode === "active" ? " active" : ""
                    }`
                  : ""}
                )
              </span>
            </p>
          </div>
        ) : null}
      </div>

      <div className="rankings-browser__grid" style={FILTER_GRID_STYLE}>
        <label className="rb-field rb-field--season">
          <span className="rb-field__label">Season</span>
          <div className="rb-select-wrap">
            <select
              className="form-select rb-field__control"
              value={guildInfo ? seasonSelectValue : ""}
              onChange={(e) => {
                const v = e.target.value;
                if (v === "active") {
                  setSeasonMode("active");
                  if (activeSeasonForServer != null) setSeason(activeSeasonForServer);
                } else {
                  const n = parseInt(v, 10);
                  if (Number.isFinite(n) && n > 0) {
                    setSeasonMode("specific");
                    setSeason(n);
                    setApiData(null);
                  }
                }
              }}
              disabled={disabled || !guildInfo || realmSeasonChoices.length === 0}
            >
              {!guildInfo ? (
                <option value="">Load guild first</option>
              ) : (
                <>
                  {activeSeasonForServer != null ? (
                    <option value="active">
                      Active (season {activeSeasonForServer})
                    </option>
                  ) : null}
                  {realmSeasonChoices
                    .filter((s) => s !== activeSeasonForServer)
                    .map((s) => (
                      <option key={s} value={String(s)}>
                        Season {s} (historical)
                      </option>
                    ))}
                </>
              )}
            </select>
            <ChevronDown
              className="rb-select-wrap__chevron"
              size={14}
              strokeWidth={2}
              aria-hidden
            />
          </div>
        </label>

        <RbSelect
          label="Raid"
          value={raidId ?? ""}
          disabled={disabled || !guildInfo || raids.length === 0}
          onChange={(e) => {
            const v = parseInt(e.target.value, 10);
            setRaidId(v);
            const r = raids.find((x) => x.value === v);
            setRaidName(r?.label ?? "");
            setApiData(null);
          }}
        >
          {raids.length === 0 && <option value="">Loading…</option>}
          {raids.map((r) => (
            <option key={r.value} value={r.value}>
              {r.label}
            </option>
          ))}
        </RbSelect>

        <RbSelect
          label="Difficulty"
          value={difficulty}
          disabled={disabled || !guildInfo}
          onChange={(e) => {
            setDifficulty(e.target.value);
            setApiData(null);
          }}
        >
          {DIFFICULTIES.map((d) => (
            <option key={d.value} value={d.value}>
              {d.label}
            </option>
          ))}
        </RbSelect>

        <RbSelect
          label="Ladder"
          value={ladder}
          disabled={disabled || !guildInfo}
          onChange={(e) => {
            setLadder(e.target.value);
            setApiData(null);
          }}
        >
          {LADDERS.map((l) => (
            <option key={l.value} value={l.value}>
              {l.label}
            </option>
          ))}
        </RbSelect>
      </div>

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
          {loading ? "Loading…" : "Get guild rankings"}
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

      {apiData && apiData.rankings.length > 0 && (
        <CharacterRankingsTable
          rankings={apiData.rankings}
          bossOrder={bossOrder}
          theme={theme}
        />
      )}

      {apiData && apiData.rankings.length === 0 && (
        <p className="rankings-browser__note">
          No ranking data for this guild and filter combination.
        </p>
      )}
    </section>
  );
}
