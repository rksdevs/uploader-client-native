import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { BarChart3, ChevronDown, ChevronUp, Crown, Shield, Swords } from "lucide-react";
import {
  BrowseAddonRankingsJSON,
  CommitAddonRankingsJSON,
  FetchLeaderboardFilterBossesJSON,
  FetchLeaderboardFilterDifficultiesJSON,
  FetchLeaderboardFilterRaidsJSON,
  FetchLeaderboardSeasonsConfigJSON,
} from "../../wailsjs/go/main/App";
import { toast } from "sonner";
import { difficultyOptionLabel } from "../lib/difficultyLabels";
import { raidBossOrderForServer } from "../lib/raidBossOrder";
import {
  getClassColor,
  getPercentileColor,
  specPortraitPath,
  toSlug,
} from "../lib/leaderboardVisuals";

type ExportMeta = {
  syncMode?: string;
  serverId?: number;
  bucketCap?: number;
  performanceSlice?: Record<string, unknown> | null;
  performanceSliceSummary?: string;
  fetchedPerfRawRows?: number;
  pointsV2?: boolean;
  pointsSliceSummary?: string;
};

type AddonBrowseResponse = {
  realm?: string;
  season?: number;
  pointsCount?: number;
  performanceCount?: number;
  performanceRows?: Record<string, unknown>[];
  rows?: Record<string, unknown>[];
  pointsV2?: boolean;
  pointsSliceSummary?: string;
  filters?: Record<string, unknown>;
  performanceFilters?: Record<string, unknown>;
  exportMeta?: ExportMeta;
};

/** When switching Type in the uploader, the API returns only one slice; keep the other from the last merged state. */
function mergeAddonBrowseResponses(
  prev: AddonBrowseResponse | null,
  next: AddonBrowseResponse
): AddonBrowseResponse {
  if (!prev) return next;
  const rPrev = (prev.realm ?? "").trim();
  const rNext = (next.realm ?? "").trim();
  if (rPrev && rNext && rPrev !== rNext) return next;
  const sPrev = prev.season;
  const sNext = next.season;
  if (sPrev != null && sNext != null && sPrev !== sNext) return next;

  const merged: AddonBrowseResponse = { ...next };
  merged.exportMeta = { ...(next.exportMeta ?? {}) };

  const nextPerf = next.performanceRows?.length ?? 0;
  const prevPerf = prev.performanceRows?.length ?? 0;
  if (nextPerf === 0 && prevPerf > 0) {
    merged.performanceRows = prev.performanceRows;
    merged.performanceFilters = prev.performanceFilters ?? next.performanceFilters;
    if (!merged.exportMeta!.performanceSliceSummary && prev.exportMeta?.performanceSliceSummary) {
      merged.exportMeta!.performanceSliceSummary = prev.exportMeta.performanceSliceSummary;
    }
    merged.performanceCount = merged.performanceRows?.length ?? 0;
  }

  const nextPts = next.rows?.length ?? 0;
  const prevPts = prev.rows?.length ?? 0;
  if (nextPts === 0 && prevPts > 0) {
    merged.rows = prev.rows;
    merged.filters = prev.filters ?? next.filters;
    merged.pointsV2 = Boolean(prev.pointsV2 ?? next.pointsV2);
    merged.pointsSliceSummary = prev.pointsSliceSummary ?? next.pointsSliceSummary;
    merged.exportMeta!.pointsV2 = Boolean(next.exportMeta?.pointsV2 || prev.exportMeta?.pointsV2);
    if (!merged.exportMeta!.pointsSliceSummary && prev.exportMeta?.pointsSliceSummary) {
      merged.exportMeta!.pointsSliceSummary = prev.exportMeta.pointsSliceSummary;
    }
    merged.pointsCount = merged.rows?.length ?? 0;
  }

  if (merged.performanceCount == null) merged.performanceCount = merged.performanceRows?.length ?? 0;
  if (merged.pointsCount == null) merged.pointsCount = merged.rows?.length ?? 0;
  return merged;
}

interface RankingsBrowserProps {
  selectedServer: string;
  serverNumericId: number | null;
  disabled: boolean;
  wowDirectory: string;
  /** Used for class name contrast on dark vs light cards. */
  theme?: "light" | "dark";
}

type FilterNum = { label: string; value: number };
type FilterStr = { label: string; value: string };

type SeasonsConfigResponse = {
  active: Record<string, number>;
  historical: Record<string, number[]>;
};

function parseFilterNumJSON(json: string): FilterNum[] {
  let data: unknown;
  try {
    data = JSON.parse(json);
  } catch {
    return [];
  }
  if (!Array.isArray(data)) return [];
  const out: FilterNum[] = [];
  for (const row of data) {
    if (!row || typeof row !== "object") continue;
    const o = row as Record<string, unknown>;
    const lab = o.label ?? o.Label;
    const val = o.value ?? o.Value;
    const label = typeof lab === "string" ? lab : String(lab ?? "");
    const n = typeof val === "number" ? val : parseInt(String(val), 10);
    if (!label.trim() || !Number.isFinite(n) || n <= 0) continue;
    out.push({ label: label.trim(), value: n });
  }
  return out;
}

function parseFilterStrJSON(json: string): FilterStr[] {
  let data: unknown;
  try {
    data = JSON.parse(json);
  } catch {
    return [];
  }
  if (!Array.isArray(data)) return [];
  const out: FilterStr[] = [];
  for (const row of data) {
    if (!row || typeof row !== "object") continue;
    const o = row as Record<string, unknown>;
    const val = o.value ?? o.Value;
    const value = String(val ?? "").trim();
    if (!value) continue;
    out.push({ label: difficultyOptionLabel(value), value });
  }
  return out;
}

const CLASSES = [
  "All",
  "Death Knight",
  "Druid",
  "Hunter",
  "Mage",
  "Paladin",
  "Priest",
  "Rogue",
  "Shaman",
  "Warlock",
  "Warrior",
] as const;

const SPECS_BY_CLASS: Record<string, string[]> = {
  All: ["All"],
  Mage: ["All", "Fire", "Arcane", "Frost"],
  Warrior: ["All", "Arms", "Fury", "Protection"],
  Rogue: ["All", "Assassination", "Combat", "Subtlety"],
  Paladin: ["All", "Holy", "Protection", "Retribution"],
  Priest: ["All", "Discipline", "Holy", "Shadow"],
  Warlock: ["All", "Affliction", "Demonology", "Destruction"],
  Hunter: ["All", "Beast Mastery", "Marksmanship", "Survival"],
  "Death Knight": ["All", "Blood", "Frost", "Unholy"],
  Druid: ["All", "Balance", "Feral Combat", "Restoration"],
  Shaman: ["All", "Elemental", "Enhancement", "Restoration"],
};

const POINTS_V2_DIFFICULTIES: { value: string; label: string }[] = [
  { value: "TEN_NM", label: difficultyOptionLabel("TEN_NM") },
  { value: "TEN_HC", label: difficultyOptionLabel("TEN_HC") },
  { value: "TWENTY_FIVE_NM", label: difficultyOptionLabel("TWENTY_FIVE_NM") },
  { value: "TWENTY_FIVE_HC", label: difficultyOptionLabel("TWENTY_FIVE_HC") },
  { value: "OVERALL", label: "Overall (weighted)" },
];

const POINTS_PHASE_OPTIONS: { label: string; value: number }[] = [
  { label: "All phases", value: 0 },
  { label: "Phase 1 (Naxx…)", value: 1 },
  { label: "Phase 2 (Ulduar)", value: 2 },
  { label: "Phase 3 (ToC…)", value: 3 },
  { label: "Phase 4 (ICC)", value: 4 },
];

const POINTS_LADDER_OPTIONS: { value: string; label: string }[] = [
  { value: "HARDCORE", label: "Hardcore" },
  { value: "COMPETITIVE", label: "Competitive" },
  { value: "REGULAR", label: "Regular" },
];

const POINTS_ROLE_FILTER_OPTIONS: { value: string; label: string }[] = [
  { value: "ALL", label: "All" },
  { value: "DPS", label: "DPS" },
  { value: "HEAL", label: "Heal" },
];

function sortBossesForRaid(
  bosses: FilterNum[] | undefined,
  raidLabel: string | undefined,
  serverEnum: string
): FilterNum[] {
  if (!bosses?.length) return bosses ?? [];
  const orderMap = raidBossOrderForServer(serverEnum);
  const order = raidLabel ? orderMap[raidLabel] : undefined;
  if (!order?.length) return [...bosses];
  const idx = (name: string) => {
    const i = order.indexOf(name);
    return i === -1 ? 999 : i;
  };
  return [...bosses].sort((a, b) => idx(a.label) - idx(b.label));
}

function readPerformanceRow(row: Record<string, unknown>) {
  const str = (v: unknown) => (typeof v === "string" ? v : v != null ? String(v) : "");
  const num = (v: unknown) => {
    if (typeof v === "number") return v;
    const n = parseFloat(String(v));
    return Number.isFinite(n) ? n : 0;
  };
  const cr = row.categoryRank;
  let categoryRank: number | null = null;
  if (typeof cr === "number" && Number.isFinite(cr)) categoryRank = cr;
  else if (cr != null) {
    const n = parseInt(String(cr), 10);
    if (Number.isFinite(n)) categoryRank = n;
  }
  return {
    key: str(row.key) || `${str(row.playerName)}-${str(row.ladder)}-${str(row.playerSpec)}`,
    playerName: str(row.playerName),
    playerClass: str(row.playerClass),
    playerSpec: str(row.playerSpec),
    role: str(row.role),
    ladder: str(row.ladder),
    difficulty: str(row.difficulty),
    amount: num(row.amount),
    percentile: num(row.percentile),
    categoryRank,
  };
}

function readPointsV2Row(row: Record<string, unknown>) {
  const str = (v: unknown) => (typeof v === "string" ? v : v != null ? String(v) : "");
  const num = (v: unknown) => {
    if (typeof v === "number") return v;
    const n = parseFloat(String(v));
    return Number.isFinite(n) ? n : 0;
  };
  const cr = row.categoryRank;
  let categoryRank: number | null = null;
  if (typeof cr === "number" && Number.isFinite(cr)) categoryRank = cr;
  else if (cr != null) {
    const n = parseInt(String(cr), 10);
    if (Number.isFinite(n)) categoryRank = n;
  }
  return {
    key: str(row.key) || `${str(row.playerName)}-${str(row.playerSpec)}`,
    playerName: str(row.playerName),
    playerClass: str(row.playerClass),
    playerSpec: str(row.playerSpec),
    role: str(row.role),
    points: num(row.points),
    specPct: num(row.specPercentileV2),
    classPct: num(row.classPercentileV2),
    rolePct: num(row.rolePercentileV2),
    categoryRank,
  };
}

function SpecPortrait({ playerClass, playerSpec }: { playerClass: string; playerSpec: string }) {
  const classJpg = `/icons/classes/${toSlug(playerClass) || "Unknown"}.jpg`;
  const primary = specPortraitPath(playerClass, playerSpec);
  const [src, setSrc] = useState(primary);
  useEffect(() => {
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

function LadderCell({ ladder }: { ladder: string }) {
  const v = ladder.trim();
  if (v === "Hardcore") {
    return (
      <span className="rankings-browser__ladder-pill" title="Hardcore">
        <Crown size={13} strokeWidth={2} className="rankings-browser__ladder-ico rankings-browser__ladder-ico--hc" />
        HC
      </span>
    );
  }
  if (v === "Competitive") {
    return (
      <span className="rankings-browser__ladder-pill" title="Competitive">
        <Swords size={13} strokeWidth={2} className="rankings-browser__ladder-ico rankings-browser__ladder-ico--cp" />
        CP
      </span>
    );
  }
  if (v === "Regular") {
    return (
      <span className="rankings-browser__ladder-pill" title="Regular">
        <Shield size={13} strokeWidth={2} className="rankings-browser__ladder-ico rankings-browser__ladder-ico--rg" />
        RG
      </span>
    );
  }
  return <span className="rankings-browser__ladder-fallback">{v || "—"}</span>;
}

function PercentilePerfCell({ percentile }: { percentile: number }) {
  const pct = Number.isFinite(percentile) ? Math.min(100, Math.max(0, percentile)) : 0;
  const color = getPercentileColor(pct);
  return (
    <div className="rankings-browser__perf-bar-cell">
      <span className="rankings-browser__perf-pct" style={{ color }}>
        {pct.toFixed(1)}%
      </span>
      <div className="rankings-browser__bar-track" aria-hidden>
        <div
          className="rankings-browser__bar-fill"
          style={{
            width: `${Math.round(pct)}%`,
            backgroundColor: color,
          }}
        />
      </div>
    </div>
  );
}

const RankingsBrowser: React.FC<RankingsBrowserProps> = ({
  selectedServer,
  serverNumericId,
  disabled,
  wowDirectory,
  theme = "light",
}) => {
  /** API: `performance` | `points_v2` — labels in UI are human-readable. */
  const [leaderboardSync, setLeaderboardSync] = useState<"performance" | "points_v2">("performance");

  /**
   * Season: "active" = omit season on filter APIs & pass 0 to BrowseAddonRankingsJSON so the
   * server uses latest season for that realm (`uploadControllerV5`: requestedSeason || server.seasons[0]...).
   */
  const [seasonMode, setSeasonMode] = useState<"active" | "specific">("active");
  const [specificSeason, setSpecificSeason] = useState<number>(4);

  const [seasonsConfig, setSeasonsConfig] = useState<SeasonsConfigResponse | null>(null);

  /** Must be a real DB server id (>0). Fallback uploader entries use id 0 and cannot load filters. */
  const serverOk = serverNumericId != null && serverNumericId > 0;

  const [raids, setRaids] = useState<FilterNum[]>([]);
  const [raidsLoading, setRaidsLoading] = useState(false);
  const [selectedRaidId, setSelectedRaidId] = useState<number | null>(null);

  const [difficulties, setDifficulties] = useState<FilterStr[]>([]);
  const [difficultiesLoading, setDifficultiesLoading] = useState(false);
  const [difficulty, setDifficulty] = useState("");

  const [bosses, setBosses] = useState<FilterNum[]>([]);
  const [bossesLoading, setBossesLoading] = useState(false);
  const [selectedBossId, setSelectedBossId] = useState<number | null>(null);

  const [role, setRole] = useState<string>("DPS");
  const [ladder, setLadder] = useState<string>("Regular");
  const [className, setClassName] = useState<string>("All");
  const [spec, setSpec] = useState<string>("All");
  const [bucketCap, setBucketCap] = useState<string>("50");

  const [pointsTimeframe, setPointsTimeframe] = useState<"seasonal" | "alltime">("seasonal");
  const [pointsPhase, setPointsPhase] = useState(0);
  const [pointsDifficulty, setPointsDifficulty] = useState("OVERALL");
  const [pointsLadder, setPointsLadder] = useState<string>("HARDCORE");
  const [pointsRoleFilter, setPointsRoleFilter] = useState<string>("ALL");

  const [loading, setLoading] = useState(false);
  const [committing, setCommitting] = useState(false);
  const [payloadJson, setPayloadJson] = useState<string | null>(null);
  const [parsed, setParsed] = useState<AddonBrowseResponse | null>(null);
  const parsedRef = useRef<AddonBrowseResponse | null>(null);
  useEffect(() => {
    parsedRef.current = parsed;
  }, [parsed]);
  /** When true, intro + filter grid are hidden so the table uses most of the card. */
  const [filtersCollapsed, setFiltersCollapsed] = useState(false);

  const seasonQueryParam = useMemo(() => {
    if (seasonMode !== "specific") return undefined;
    if (!Number.isFinite(specificSeason) || specificSeason <= 0) return undefined;
    return specificSeason;
  }, [seasonMode, specificSeason]);

  const browseSeasonArg = useMemo(() => {
    if (seasonMode !== "specific") return 0;
    if (!Number.isFinite(specificSeason) || specificSeason <= 0) return 0;
    return specificSeason;
  }, [seasonMode, specificSeason]);

  const activeSeasonForServer = useMemo(() => {
    if (!serverOk || !seasonsConfig?.active) return null;
    return seasonsConfig.active[String(serverNumericId)] ?? null;
  }, [serverOk, serverNumericId, seasonsConfig]);

  const seasonChoices = useMemo(() => {
    if (!serverOk || !seasonsConfig) return [];
    const sid = String(serverNumericId);
    const hist = seasonsConfig.historical[sid] ?? [];
    const active = seasonsConfig.active[sid];
    const set = new Set<number>();
    if (active != null) set.add(active);
    hist.forEach((n) => set.add(n));
    return Array.from(set).sort((a, b) => b - a);
  }, [serverOk, serverNumericId, seasonsConfig]);

  useEffect(() => {
    if (seasonMode === "specific" && seasonChoices.length && !seasonChoices.includes(specificSeason)) {
      setSpecificSeason(seasonChoices[0]);
    }
  }, [seasonMode, seasonChoices, specificSeason]);

  useEffect(() => {
    let cancelled = false;
    FetchLeaderboardSeasonsConfigJSON()
      .then((json) => {
        if (cancelled) return;
        try {
          setSeasonsConfig(JSON.parse(json) as SeasonsConfigResponse);
        } catch {
          setSeasonsConfig(null);
        }
      })
      .catch(() => {
        if (!cancelled) setSeasonsConfig(null);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!serverOk) {
      setRaids([]);
      setSelectedRaidId(null);
      return;
    }
    const seasonArg = seasonQueryParam ?? 0;
    let cancelled = false;
    setRaidsLoading(true);
    FetchLeaderboardFilterRaidsJSON(serverNumericId, seasonArg)
      .then((json) => {
        if (cancelled) return;
        const data = parseFilterNumJSON(json);
        setRaids(data);
        setSelectedRaidId((prev) => {
          if (prev != null && data.some((r) => r.value === prev)) return prev;
          return data[0]?.value ?? null;
        });
      })
      .catch((e: unknown) => {
        if (!cancelled) {
          console.error(e);
          toast.error("Failed to load raids", { description: String(e) });
          setRaids([]);
          setSelectedRaidId(null);
        }
      })
      .finally(() => {
        if (!cancelled) setRaidsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [serverOk, serverNumericId, seasonQueryParam]);

  useEffect(() => {
    if (!serverOk || selectedRaidId == null) {
      setDifficulties([]);
      setDifficulty("");
      return;
    }
    const seasonArg = seasonQueryParam ?? 0;
    let cancelled = false;
    setDifficultiesLoading(true);
    FetchLeaderboardFilterDifficultiesJSON(serverNumericId, selectedRaidId, seasonArg)
      .then((json) => {
        if (cancelled) return;
        const list = parseFilterStrJSON(json);
        setDifficulties(list);
        setDifficulty((prev) => {
          if (prev && list.some((d) => d.value === prev)) return prev;
          return list[0]?.value ?? "";
        });
      })
      .catch((e: unknown) => {
        if (!cancelled) {
          console.error(e);
          toast.error("Failed to load difficulties", { description: String(e) });
          setDifficulties([]);
          setDifficulty("");
        }
      })
      .finally(() => {
        if (!cancelled) setDifficultiesLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [serverOk, serverNumericId, selectedRaidId, seasonQueryParam]);

  const currentRaidLabel = useMemo(() => {
    const r = raids.find((x) => x.value === selectedRaidId);
    return r?.label;
  }, [raids, selectedRaidId]);

  useEffect(() => {
    if (!serverOk || selectedRaidId == null || !difficulty) {
      setBosses([]);
      setSelectedBossId(null);
      return;
    }
    const seasonArg = seasonQueryParam ?? 0;
    let cancelled = false;
    setBossesLoading(true);
    FetchLeaderboardFilterBossesJSON(serverNumericId, selectedRaidId, difficulty, seasonArg)
      .then((json) => {
        if (cancelled) return;
        const raw = parseFilterNumJSON(json);
        const sorted = sortBossesForRaid(raw, currentRaidLabel, selectedServer);
        setBosses(sorted);
        setSelectedBossId((prev) => {
          if (prev != null && sorted.some((b) => b.value === prev)) return prev;
          return sorted[0]?.value ?? null;
        });
      })
      .catch((e: unknown) => {
        if (!cancelled) {
          console.error(e);
          toast.error("Failed to load bosses", { description: String(e) });
          setBosses([]);
          setSelectedBossId(null);
        }
      })
      .finally(() => {
        if (!cancelled) setBossesLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [
    serverOk,
    serverNumericId,
    selectedRaidId,
    difficulty,
    seasonQueryParam,
    currentRaidLabel,
    selectedServer,
  ]);

  const specOptions = useMemo(() => {
    return SPECS_BY_CLASS[className] ?? ["All"];
  }, [className]);

  useEffect(() => {
    if (!specOptions.includes(spec)) setSpec("All");
  }, [specOptions, spec]);

  const tableRows =
    leaderboardSync === "points_v2" ? (parsed?.rows ?? []) : (parsed?.performanceRows ?? []);
  const displayRows = tableRows.slice(0, 400);

  const buildExtraQuery = useCallback((): string => {
    if (leaderboardSync === "points_v2") {
      const lad = pointsLadder.trim().toUpperCase();
      if (lad !== "HARDCORE" && lad !== "COMPETITIVE" && lad !== "REGULAR") {
        throw new Error("Select a points ladder (Hardcore, Competitive, or Regular).");
      }
      if (!pointsDifficulty.trim()) {
        throw new Error("Select a difficulty (or Overall).");
      }
      const o: Record<string, unknown> = {
        syncMode: "points_v2",
        ladder: lad,
        timeframe: pointsTimeframe,
        difficulty: pointsDifficulty.trim(),
      };
      if (serverNumericId != null) o.serverId = serverNumericId;
      if (pointsPhase > 0) o.phase = pointsPhase;
      const c = className.trim();
      if (c && c !== "All") o.class = c;
      const s = spec.trim();
      if (s && s !== "All") o.spec = s;
      if (pointsRoleFilter === "DPS" || pointsRoleFilter === "HEAL") {
        o.pointsRole = pointsRoleFilter;
      }
      const cap = parseInt(bucketCap, 10);
      if (!Number.isNaN(cap) && cap > 0) o.bucketCap = cap;
      return JSON.stringify(o);
    }
    if (selectedBossId == null || !difficulty) {
      throw new Error("Select raid, difficulty, and boss first.");
    }
    const o: Record<string, unknown> = {
      syncMode: "performance",
      bossId: selectedBossId,
      difficulty,
      role: role.toUpperCase(),
    };
    if (serverNumericId != null) o.serverId = serverNumericId;
    if (selectedRaidId != null) o.raidId = selectedRaidId;
    if (ladder.trim()) o.ladder = ladder.trim();
    const c = className.trim();
    if (c && c !== "All") o.class = c;
    const s = spec.trim();
    if (s && s !== "All") o.spec = s;
    const cap = parseInt(bucketCap, 10);
    if (!Number.isNaN(cap) && cap > 0) o.bucketCap = cap;
    return JSON.stringify(o);
  }, [
    leaderboardSync,
    serverNumericId,
    selectedBossId,
    difficulty,
    role,
    ladder,
    className,
    spec,
    bucketCap,
    selectedRaidId,
    pointsLadder,
    pointsTimeframe,
    pointsPhase,
    pointsDifficulty,
    pointsRoleFilter,
  ]);

  const handleLoad = () => {
    if (!selectedServer && serverNumericId == null) {
      toast.error("Select a server first.");
      return;
    }
    let extra: string;
    try {
      extra = buildExtraQuery();
    } catch (e: unknown) {
      toast.error(String(e));
      return;
    }
    setLoading(true);
    setPayloadJson(null);
    BrowseAddonRankingsJSON(selectedServer, browseSeasonArg, extra)
      .then((json: string) => {
        setPayloadJson(json);
        try {
          const data = JSON.parse(json) as AddonBrowseResponse;
          const merged = mergeAddonBrowseResponses(parsedRef.current, data);
          setParsed(merged);
          toast.success(
            merged.pointsV2 || merged.exportMeta?.pointsV2
              ? `Loaded Points V2: ${merged.pointsCount ?? 0} rows`
              : `Loaded: ${merged.performanceCount ?? 0} performance rows, ${merged.pointsCount ?? 0} points rows`
          );
        } catch {
          toast.error("Could not parse API JSON");
        }
      })
      .catch((err: unknown) => {
        toast.error("Failed to load rankings", { description: String(err) });
      })
      .finally(() => setLoading(false));
  };

  const handleSendToAddon = () => {
    if (!wowDirectory.trim()) {
      toast.error("Link your WoW directory before writing src/RankingsPayload.lua.")
      return;
    }
    let extra: string;
    try {
      extra = buildExtraQuery();
    } catch (e: unknown) {
      toast.error(String(e));
      return;
    }
    setCommitting(true);
    BrowseAddonRankingsJSON(selectedServer, browseSeasonArg, extra)
      .then((json: string) => {
        setPayloadJson(json);
        let mergedJson = json;
        try {
          const data = JSON.parse(json) as AddonBrowseResponse;
          const merged = mergeAddonBrowseResponses(parsedRef.current, data);
          setParsed(merged);
          mergedJson = JSON.stringify(merged);
          setPayloadJson(mergedJson);
        } catch {
          /* Commit uses raw response if parse fails */
        }
        return CommitAddonRankingsJSON(mergedJson);
      })
      .then((msg: string) => {
        toast.success(msg, { duration: 14000 });
      })
      .catch((err: unknown) => {
        toast.error("Failed to write addon file", { description: String(err) });
      })
      .finally(() => setCommitting(false));
  };

  const filtersBusy =
    disabled ||
    loading ||
    !serverOk ||
    (leaderboardSync === "performance" &&
      (raidsLoading || difficultiesLoading || bossesLoading));

  const perfReady =
    leaderboardSync === "points_v2"
      ? serverOk && !!pointsLadder.trim() && !!pointsDifficulty.trim()
      : selectedRaidId != null && !!difficulty && selectedBossId != null && bosses.length > 0;

  return (
    <section
      className={`redesign-card rankings-browser rankings-browser--compact${
        filtersCollapsed ? " rankings-browser--filters-hidden" : ""
      }`}
    >
      <div className="card-head-row rankings-browser__head">
        <BarChart3 size={20} strokeWidth={2} className="card-head-icon" aria-hidden />
        <h2 className="rankings-browser__title" style={{ flex: 1, margin: 0 }}>
          Browse rankings
        </h2>
        <button
          type="button"
          className="btn-surface rankings-browser__filter-toggle"
          onClick={() => setFiltersCollapsed((c) => !c)}
          aria-expanded={!filtersCollapsed}
        >
          {filtersCollapsed ? (
            <>
              Show filters <ChevronDown size={16} strokeWidth={2} aria-hidden />
            </>
          ) : (
            <>
              Hide filters <ChevronUp size={16} strokeWidth={2} aria-hidden />
            </>
          )}
        </button>
      </div>

      {!filtersCollapsed ? (
        <>
          <p className="rankings-browser__lead rankings-browser__lead--tight">
            {leaderboardSync === "performance" ? (
              <>
                Same filter cascade as{" "}
                <a
                  href="https://wow-logs.co.in/leaderboard"
                  target="_blank"
                  rel="noreferrer"
                  className="rankings-browser__link"
                >
                  wow-logs.co.in/leaderboard
                </a>
                : pick raid → difficulty → boss (boss list depends on raid).{" "}
                <strong>Season:</strong> leave on &quot;Active season&quot; to match the site (server&apos;s
                current season). Choose a number only when you want a past season.
              </>
            ) : (
              <>
                Same filters as{" "}
                <a
                  href="https://wow-logs.co.in/points-leaderboard"
                  target="_blank"
                  rel="noreferrer"
                  className="rankings-browser__link"
                >
                  Points leaderboard (V2)
                </a>
                : ladder, timeframe, phase, difficulty, class/spec, role — then load rows for the in-game
                Points tab.
              </>
            )}
          </p>

          {selectedServer.trim() && !serverOk ? (
            <p className="rankings-browser__lead" style={{ marginTop: 0, color: "var(--rb-warn, #c9a227)" }}>
              This realm has no API id yet (id 0). Use <strong>Refresh servers</strong> in the uploader after
              the API is reachable so raid/difficulty/boss lists can load.
            </p>
          ) : null}

          <div className="rankings-browser__grid">
        <label className="rb-field">
          <span className="rb-field__label">Type</span>
          <select
            className="form-select rb-field__control"
            value={leaderboardSync}
            onChange={(e) => setLeaderboardSync(e.target.value as "performance" | "points_v2")}
            disabled={disabled || loading}
          >
            <option value="performance">Performance leaderboard</option>
            <option value="points_v2">Points V2 leaderboard</option>
          </select>
        </label>

        <div className="rb-field rb-field--season">
          <span className="rb-field__label">Season</span>
          <select
            className="form-select rb-field__control"
            value={seasonMode}
            onChange={(e) => setSeasonMode(e.target.value as "active" | "specific")}
            disabled={disabled || loading}
          >
            <option value="active">Active (site default)</option>
            <option value="specific">Specific season…</option>
          </select>
          {seasonMode === "specific" && (
            <select
              className="form-select rb-field__control rb-field__control--stacked"
              value={specificSeason}
              onChange={(e) => setSpecificSeason(parseInt(e.target.value, 10))}
              disabled={disabled || loading || !seasonChoices.length}
            >
              {seasonChoices.map((n) => (
                <option key={n} value={n}>
                  Season {n}
                  {activeSeasonForServer === n ? " (active on site)" : ""}
                </option>
              ))}
            </select>
          )}
          {seasonMode === "active" && activeSeasonForServer != null && (
            <p className="rb-field__microcopy">
              Active season for this realm: <strong>{activeSeasonForServer}</strong>
            </p>
          )}
        </div>

        {leaderboardSync === "performance" && (
          <>
            <label className="rb-field">
              <span className="rb-field__label">Raid</span>
              <select
                className="form-select rb-field__control"
                value={selectedRaidId ?? ""}
                onChange={(e) => {
                  const v = parseInt(e.target.value, 10);
                  setSelectedRaidId(Number.isNaN(v) ? null : v);
                }}
                disabled={filtersBusy || !raids.length}
              >
                {!raids.length ? <option value="">Loading…</option> : null}
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
                onChange={(e) => setDifficulty(e.target.value)}
                disabled={filtersBusy || !difficulties.length}
              >
                {!difficulties.length ? <option value="">Select raid first…</option> : null}
                {difficulties.map((d) => (
                  <option key={d.value} value={d.value}>
                    {difficultyOptionLabel(d.value)}
                  </option>
                ))}
              </select>
            </label>

            <label className="rb-field">
              <span className="rb-field__label">
                Boss <span className="rb-field__req">*</span>
              </span>
              <select
                className="form-select rb-field__control"
                value={selectedBossId ?? ""}
                onChange={(e) => {
                  const v = parseInt(e.target.value, 10);
                  setSelectedBossId(Number.isNaN(v) ? null : v);
                }}
                disabled={filtersBusy || !bosses.length}
              >
                {!bosses.length ? <option value="">Select difficulty first…</option> : null}
                {bosses.map((b) => (
                  <option key={b.value} value={b.value}>
                    {b.label}
                  </option>
                ))}
              </select>
            </label>

            <label className="rb-field">
              <span className="rb-field__label">
                Role <span className="rb-field__req">*</span>
              </span>
              <select
                className="form-select rb-field__control"
                value={role}
                onChange={(e) => setRole(e.target.value)}
                disabled={disabled || loading}
              >
                <option value="DPS">DPS</option>
                <option value="HEALER">Healers (HEALER)</option>
              </select>
            </label>

            <label className="rb-field">
              <span className="rb-field__label">Ladder</span>
              <select
                className="form-select rb-field__control"
                value={ladder}
                onChange={(e) => setLadder(e.target.value)}
                disabled={disabled || loading}
              >
                <option value="">All ladders</option>
                <option value="Hardcore">Hardcore</option>
                <option value="Competitive">Competitive</option>
                <option value="Regular">Regular</option>
              </select>
            </label>

            <label className="rb-field">
              <span className="rb-field__label">Class</span>
              <select
                className="form-select rb-field__control"
                value={className}
                onChange={(e) => {
                  setClassName(e.target.value);
                  setSpec("All");
                }}
                disabled={disabled || loading}
              >
                {CLASSES.map((c) => (
                  <option key={c} value={c}>
                    {c}
                  </option>
                ))}
              </select>
            </label>

            <label className="rb-field">
              <span className="rb-field__label">Spec</span>
              <select
                className="form-select rb-field__control"
                value={spec}
                onChange={(e) => setSpec(e.target.value)}
                disabled={disabled || loading || className === "All"}
              >
                {specOptions.map((s) => (
                  <option key={s} value={s}>
                    {s}
                  </option>
                ))}
              </select>
            </label>

            <label className="rb-field rb-field--full">
              <span className="rb-field__label">Rows</span>
              <input
                className="form-input rb-field__control"
                value={bucketCap}
                onChange={(e) => setBucketCap(e.target.value.replace(/[^\d]/g, ""))}
                placeholder="50"
                title="How many leaderboard rows to fetch per bucket (1–500; server clamps). Same cap trims points and performance in the addon export."
                disabled={disabled || loading}
              />
            </label>
          </>
        )}

        {leaderboardSync === "points_v2" && (
          <>
            <label className="rb-field">
              <span className="rb-field__label">
                Points ladder <span className="rb-field__req">*</span>
              </span>
              <select
                className="form-select rb-field__control"
                value={pointsLadder}
                onChange={(e) => setPointsLadder(e.target.value)}
                disabled={disabled || loading}
              >
                {POINTS_LADDER_OPTIONS.map((o) => (
                  <option key={o.value} value={o.value}>
                    {o.label}
                  </option>
                ))}
              </select>
            </label>

            <label className="rb-field">
              <span className="rb-field__label">Timeframe</span>
              <select
                className="form-select rb-field__control"
                value={pointsTimeframe}
                onChange={(e) =>
                  setPointsTimeframe(e.target.value as "seasonal" | "alltime")
                }
                disabled={disabled || loading}
              >
                <option value="seasonal">Seasonal</option>
                <option value="alltime">All time</option>
              </select>
            </label>

            <label className="rb-field">
              <span className="rb-field__label">Phase</span>
              <select
                className="form-select rb-field__control"
                value={pointsPhase}
                onChange={(e) => setPointsPhase(parseInt(e.target.value, 10) || 0)}
                disabled={disabled || loading}
              >
                {POINTS_PHASE_OPTIONS.map((p) => (
                  <option key={p.value} value={p.value}>
                    {p.label}
                  </option>
                ))}
              </select>
            </label>

            <label className="rb-field">
              <span className="rb-field__label">Difficulty total</span>
              <select
                className="form-select rb-field__control"
                value={pointsDifficulty}
                onChange={(e) => setPointsDifficulty(e.target.value)}
                disabled={disabled || loading}
              >
                {POINTS_V2_DIFFICULTIES.map((d) => (
                  <option key={d.value} value={d.value}>
                    {d.label}
                  </option>
                ))}
              </select>
            </label>

            <label className="rb-field">
              <span className="rb-field__label">Class</span>
              <select
                className="form-select rb-field__control"
                value={className}
                onChange={(e) => {
                  setClassName(e.target.value);
                  setSpec("All");
                }}
                disabled={disabled || loading}
              >
                {CLASSES.map((c) => (
                  <option key={c} value={c}>
                    {c}
                  </option>
                ))}
              </select>
            </label>

            <label className="rb-field">
              <span className="rb-field__label">Spec</span>
              <select
                className="form-select rb-field__control"
                value={spec}
                onChange={(e) => setSpec(e.target.value)}
                disabled={disabled || loading || className === "All"}
              >
                {specOptions.map((s) => (
                  <option key={s} value={s}>
                    {s}
                  </option>
                ))}
              </select>
            </label>

            <label className="rb-field">
              <span className="rb-field__label">Role filter</span>
              <select
                className="form-select rb-field__control"
                value={pointsRoleFilter}
                onChange={(e) => setPointsRoleFilter(e.target.value)}
                disabled={disabled || loading}
              >
                {POINTS_ROLE_FILTER_OPTIONS.map((o) => (
                  <option key={o.value} value={o.value}>
                    {o.label}
                  </option>
                ))}
              </select>
            </label>

            <label className="rb-field rb-field--full">
              <span className="rb-field__label">Rows</span>
              <input
                className="form-input rb-field__control"
                value={bucketCap}
                onChange={(e) => setBucketCap(e.target.value.replace(/[^\d]/g, ""))}
                placeholder="50"
                title="Max rows in this slice (1–500; server clamps). Followed players from Premium settings are kept first."
                disabled={disabled || loading}
              />
            </label>
          </>
        )}
          </div>
        </>
      ) : null}

      <div className="rankings-browser__actions">
        <button
          type="button"
          className="btn-slate-solid"
          onClick={handleLoad}
          disabled={
            disabled ||
            loading ||
            (!selectedServer && serverNumericId == null) ||
            !perfReady
          }
        >
          {loading ? "Loading…" : "Load rankings"}
        </button>
        <button
          type="button"
          className="btn-send-addon"
          onClick={handleSendToAddon}
          disabled={disabled || committing || !wowDirectory.trim() || !perfReady}
        >
          {committing ? "Writing…" : "Send to addon"}
        </button>
      </div>

      {parsed && (
        <div className="rankings-browser__table-wrap">
          {leaderboardSync === "points_v2" ? (
            <>
              <table className="rankings-browser__perf-table">
                <thead>
                  <tr>
                    <th className="rb-col-rank">#</th>
                    <th className="rb-col-player">Player</th>
                    <th className="rb-col-role">Role</th>
                    <th className="rb-col-amt">Points</th>
                    <th className="rb-col-perf">Spec %</th>
                    <th className="rb-col-perf">Class %</th>
                    <th className="rb-col-perf">Role %</th>
                  </tr>
                </thead>
                <tbody>
                  {displayRows.map((row, idx) => {
                    const r = readPointsV2Row(row);
                    const rank = r.categoryRank ?? idx + 1;
                    const nameColor = getClassColor(r.playerClass, theme);
                    return (
                      <tr key={r.key || `pv2-${idx}`}>
                        <td className="rb-col-rank">#{rank}</td>
                        <td className="rb-col-player">
                          <div className="rankings-browser__player-cell">
                            <SpecPortrait playerClass={r.playerClass} playerSpec={r.playerSpec} />
                            <span className="rankings-browser__player-name" style={{ color: nameColor }}>
                              {r.playerName}
                            </span>
                          </div>
                        </td>
                        <td className="rb-col-role">{r.role}</td>
                        <td className="rb-col-amt">{r.points.toFixed(2)}</td>
                        <td className="rb-col-perf">
                          <PercentilePerfCell percentile={r.specPct} />
                        </td>
                        <td className="rb-col-perf">
                          <PercentilePerfCell percentile={r.classPct} />
                        </td>
                        <td className="rb-col-perf">
                          <PercentilePerfCell percentile={r.rolePct} />
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
              {tableRows.length > displayRows.length && (
                <div style={{ padding: "6px", fontSize: "11px", opacity: 0.8 }}>
                  Showing first {displayRows.length} of {tableRows.length} points rows.
                </div>
              )}
            </>
          ) : (
            <>
              <table className="rankings-browser__perf-table">
                <thead>
                  <tr>
                    <th className="rb-col-rank">#</th>
                    <th className="rb-col-player">Player</th>
                    <th className="rb-col-role">Role</th>
                    <th className="rb-col-ladder">Ladder</th>
                    <th className="rb-col-diff">Diff</th>
                    <th className="rb-col-amt">{role === "HEALER" ? "HPS" : "DPS"}</th>
                    <th className="rb-col-perf">Performance</th>
                  </tr>
                </thead>
                <tbody>
                  {displayRows.map((row, idx) => {
                    const r = readPerformanceRow(row);
                    const rank = r.categoryRank ?? idx + 1;
                    const nameColor = getClassColor(r.playerClass, theme);
                    return (
                      <tr key={r.key || `row-${idx}`}>
                        <td className="rb-col-rank">#{rank}</td>
                        <td className="rb-col-player">
                          <div className="rankings-browser__player-cell">
                            <SpecPortrait playerClass={r.playerClass} playerSpec={r.playerSpec} />
                            <span className="rankings-browser__player-name" style={{ color: nameColor }}>
                              {r.playerName}
                            </span>
                          </div>
                        </td>
                        <td className="rb-col-role">{r.role}</td>
                        <td className="rb-col-ladder">
                          <LadderCell ladder={r.ladder} />
                        </td>
                        <td className="rb-col-diff">
                          {r.difficulty ? difficultyOptionLabel(r.difficulty) : ""}
                        </td>
                        <td className="rb-col-amt">{Number(r.amount).toFixed(0)}</td>
                        <td className="rb-col-perf">
                          <PercentilePerfCell percentile={r.percentile} />
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
              {tableRows.length > displayRows.length && (
                <div style={{ padding: "6px", fontSize: "11px", opacity: 0.8 }}>
                  Showing first {displayRows.length} of {tableRows.length} performance rows.
                </div>
              )}
            </>
          )}
        </div>
      )}
    </section>
  );
};

export default RankingsBrowser;
