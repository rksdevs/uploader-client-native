import { useEffect, useMemo, useState } from "react";
import { toast } from "sonner";
import { CheckCircle2, Copy, FolderOpen, RefreshCw, UploadCloud } from "lucide-react";
import {
  CreateAsCleanCopy,
  GetAsLocations,
  GetAsUploadStatus,
  KnownAsRuns,
  OpenAsFolder,
  ScanAsLog,
  SelectAsAddonFile,
  SelectAsCopiesDir,
  SelectAsGameFolder,
  SelectAsLogFile,
  SelectAsStatsFile,
  SetAsLocation,
  UploadAsRuns,
} from "../../wailsjs/go/main/App";
import { main } from "../../wailsjs/go/models";
import { BrowserOpenURL } from "../../wailsjs/runtime";
import { Badge, Button, Callout, Card, EmptyState, Input, PageHeader, Switch } from "../ui";
import { useApp } from "../state/AppContext";

// Older builds remembered only the log, in the browser store. It moves into the app's config once.
const LEGACY_LOG_KEY = "asLogPath";

type Run = main.AsScanResult["runs"][number];
type LocationKind = "log" | "addon" | "stats" | "copies";

function duration(ms: number) {
  const total = Math.floor(ms / 1000);
  return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, "0")}`;
}

function fileName(path: string) {
  return path.split(/[\\/]/).pop() ?? path;
}

function errorText(error: unknown) {
  return String(error).replace(/^Error:\s*/, "");
}

/**
 * Opens the website in the player's default browser, as a new tab in its last window, where they are
 * signed in. A target="_blank" link would open a bare app window instead, with no site sign-in.
 */
function openOnSite(url: string) {
  BrowserOpenURL(url);
}

// Switch carries its label for screen readers only, so the text is shown beside it, as on other screens.
function SwitchRow({ label, note, checked, disabled, onChange }: {
  label: string;
  note?: string;
  checked: boolean;
  disabled?: boolean;
  onChange: (next: boolean) => void;
}) {
  return (
    <label className="flex items-center justify-between gap-3 text-sm">
      <span className="min-w-0">
        <span className="block">{label}</span>
        {note && <span className="block text-xs text-muted-foreground">{note}</span>}
      </span>
      <Switch label={label} checked={checked} disabled={disabled} onChange={onChange} />
    </label>
  );
}

function LocationRow({ label, path, found, chosen, folder, onChange, onReset }: {
  label: string;
  path: string;
  found?: boolean;
  chosen?: boolean;
  folder?: boolean;
  onChange: () => void;
  onReset?: () => void;
}) {
  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5 border-b border-border py-2.5 last:border-b-0">
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2 text-sm font-medium">
          {label}
          {!folder && (found ? <Badge tone="success">Found</Badge> : <Badge tone="warning">Not found</Badge>)}
          {chosen && <Badge>Chosen</Badge>}
        </div>
        <div className="truncate font-mono text-xs text-muted-foreground" title={path}>
          {path || "Not set"}
        </div>
      </div>
      <div className="flex gap-1.5">
        {chosen && onReset && (
          <Button size="sm" variant="ghost" onClick={onReset}>Use game folder</Button>
        )}
        <Button size="sm" variant="outline" onClick={onChange}>Change</Button>
      </div>
    </div>
  );
}

function RunRow({ run, match, checked, short, uploadedUrl, onToggle }: {
  run: Run;
  match?: main.AsRunMatch;
  checked: boolean;
  short: boolean;
  /** The run's page when this player already uploaded it. */
  uploadedUrl?: string;
  onToggle: () => void;
}) {
  return (
    <label className="flex cursor-pointer items-start gap-3 rounded-control border border-border p-3 hover:bg-muted/50">
      <input type="checkbox" className="mt-1 size-4 accent-brand" checked={checked} onChange={onToggle} />
      <span className="min-w-0 flex-1 space-y-0.5">
        <span className="flex flex-wrap items-center gap-2 text-sm font-medium">
          {run.start}
          {uploadedUrl && (
            <>
              <Badge tone="success">Uploaded</Badge>
              <button
                type="button"
                className="cursor-pointer text-xs font-normal text-brand underline-offset-2 hover:underline"
                onClick={(event) => {
                  // Inside the row's label: don't also tick the run.
                  event.preventDefault();
                  event.stopPropagation();
                  openOnSite(uploadedUrl);
                }}
              >
                View run
              </button>
            </>
          )}
        </span>
        <span className="block truncate text-xs text-muted-foreground">
          {run.topSpells?.map((spell) => spell.name).join(", ") || "No spells"}
        </span>
        <span className="flex flex-wrap gap-x-3 font-mono text-xs text-muted-foreground">
          <span>{duration(run.activeMs)} active</span>
          <span>{run.kills.toLocaleString()} kills</span>
          <span>{(run.damage + run.petDamage).toLocaleString()} damage</span>
        </span>
        {short && <span className="block text-xs text-gold">Under a minute of play, so it won't get a report.</span>}
        {uploadedUrl && checked && (
          <span className="block text-xs text-muted-foreground">
            Uploading it again updates your existing report, for example with addon data you didn't send before.
          </span>
        )}
      </span>
      <span className="flex flex-col items-end gap-1">
        <Badge tone={run.endReason === "death" ? "neutral" : "brand"}>
          {run.endReason === "death" ? "Death" : "No death"}
        </Badge>
        {match?.addon && <Badge tone="success">WoW Logs AS</Badge>}
        {match?.stats && <Badge tone="gold">Azeroth Stats</Badge>}
      </span>
    </label>
  );
}

export function SurvivorsScreen() {
  const { account, signIn, siteUrl } = useApp();
  const [locations, setLocations] = useState<main.AsLocations | null>(null);
  const [scan, setScan] = useState<main.AsScanResult | null>(null);
  const [selected, setSelected] = useState<Set<number>>(new Set());
  const [sendAddon, setSendAddon] = useState(true);
  const [sendStats, setSendStats] = useState(true);
  const [heroName, setHeroName] = useState("");
  const [isPrivate, setIsPrivate] = useState(true);
  const [scanning, setScanning] = useState(false);
  const [uploadId, setUploadId] = useState<number | null>(null);
  const [uploadStartedAt, setUploadStartedAt] = useState<number | null>(null);
  const [status, setStatus] = useState<main.AsUploadStatus | null>(null);
  const [copying, setCopying] = useState(false);
  const [copy, setCopy] = useState<main.AsCopyResult | null>(null);
  // Runs of this log already on the account: run index -> publicId.
  const [known, setKnown] = useState<Map<number, string>>(new Map());

  // A failed check marks nothing, so a run is never hidden by mistake.
  async function checkKnown(): Promise<Map<number, string>> {
    try {
      return new Map((await KnownAsRuns()).map((run) => [run.index, run.publicId]));
    } catch {
      return new Map();
    }
  }

  async function open(path: string) {
    if (!path) return;
    setScanning(true);
    setStatus(null);
    setUploadId(null);
    setCopy(null);
    try {
      const result = await ScanAsLog(path);
      const uploaded = await checkKnown();
      setScan(result);
      setKnown(uploaded);
      // Runs already uploaded start unticked, like short ones.
      setSelected(new Set((result.runs ?? [])
        .filter((run) => run.activeMs >= result.minActiveMs && !uploaded.has(run.index))
        .map((run) => run.index)));
    } catch (error) {
      setScan(null);
      toast.error("Couldn't read this log", { description: errorText(error) });
    } finally {
      setScanning(false);
    }
  }

  // Locations changed: show them, and read the log again so matches use the new saves.
  async function applyLocations(next: main.AsLocations) {
    setLocations(next);
    if (next.logFound) await open(next.logPath);
    else setScan(null);
  }

  useEffect(() => {
    (async () => {
      let current = await GetAsLocations();
      const legacy = localStorage.getItem(LEGACY_LOG_KEY);
      if (!current.logPath && legacy) current = await SetAsLocation("log", legacy);
      localStorage.removeItem(LEGACY_LOG_KEY);
      await applyLocations(current);
    })().catch((error) => toast.error("Couldn't load your locations", { description: errorText(error) }));
  }, []);

  // Polls until the worker finishes. It takes seconds, so a simple timer is enough.
  useEffect(() => {
    if (uploadId == null) return;
    const timer = setInterval(() => {
      GetAsUploadStatus(uploadId)
        .then((next) => {
          setStatus(next);
          if (next.status === "DONE" || next.status === "FAILED") setUploadId(null);
          // The runs just stored now show as uploaded, and leave the ticked set.
          if (next.status === "DONE") {
            void checkKnown().then((uploaded) => {
              setKnown(uploaded);
              setSelected((prev) => new Set([...prev].filter((index) => !uploaded.has(index))));
            });
          }
        })
        .catch(() => setUploadId(null));
    }, 2000);
    return () => clearInterval(timer);
  }, [uploadId]);

  const runs = scan?.runs ?? [];
  const matches = useMemo(() => new Map((scan?.matches ?? []).map((match) => [match.index, match])), [scan]);
  const chosen = useMemo(() => [...selected].sort((a, b) => a - b), [selected]);
  const chosenAddon = chosen.filter((index) => matches.get(index)?.addon).length;
  const chosenStats = chosen.filter((index) => matches.get(index)?.stats).length;
  const withAddon = runs.filter((run) => matches.get(run.index)?.addon).length;
  const withStats = runs.filter((run) => matches.get(run.index)?.stats).length;
  const uploadedCount = runs.filter((run) => known.has(run.index)).length;
  const finished = status?.status === "DONE" || status?.status === "FAILED";
  const uploading = uploadId != null;
  // Re-evaluated on every 2 s status poll. Matches the website's one-minute "taking longer" note.
  const waitedLong = uploading && status?.status !== "PROCESSING" && uploadStartedAt != null && Date.now() - uploadStartedAt >= 60_000;

  function toggle(index: number) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(index)) next.delete(index);
      else next.add(index);
      return next;
    });
  }

  function selectWhere(keep: (run: Run) => boolean) {
    setSelected(new Set(runs.filter(keep).map((run) => run.index)));
  }

  async function change(kind: LocationKind) {
    const pickers: Record<LocationKind, () => Promise<string>> = {
      log: SelectAsLogFile,
      addon: SelectAsAddonFile,
      stats: SelectAsStatsFile,
      copies: SelectAsCopiesDir,
    };
    try {
      const path = await pickers[kind]();
      if (!path) return;
      const next = await SetAsLocation(kind, path);
      if (kind === "copies") setLocations(next);
      else await applyLocations(next);
    } catch (error) {
      toast.error("Couldn't change this location", { description: errorText(error) });
    }
  }

  async function reset(kind: LocationKind) {
    try {
      await applyLocations(await SetAsLocation(kind, ""));
    } catch (error) {
      toast.error("Couldn't change this location", { description: errorText(error) });
    }
  }

  async function useGameFolder() {
    try {
      await applyLocations(await SelectAsGameFolder());
    } catch (error) {
      toast.error("Couldn't use this folder", { description: errorText(error) });
    }
  }

  async function upload() {
    try {
      const created = await UploadAsRuns(
        main.AsUploadRequest.createFrom({ indexes: chosen, private: isPrivate, heroName, sendAddon, sendStats }),
      );
      setStatus(created);
      setUploadStartedAt(Date.now());
      setUploadId(created.id);
    } catch (error) {
      toast.error("Upload failed", { description: errorText(error) });
    }
  }

  async function createCopy() {
    setCopying(true);
    try {
      setCopy(await CreateAsCleanCopy(
        main.AsCopyRequest.createFrom({ indexes: chosen, includeAddon: sendAddon, includeStats: sendStats }),
      ));
    } catch (error) {
      toast.error("Couldn't create the copy", { description: errorText(error) });
    } finally {
      setCopying(false);
    }
  }

  return (
    <div className="space-y-4">
      <PageHeader
        title="Azeroth Survivors"
        description="Choose which runs to upload. The rest of your log and addon data stays on this computer."
        action={
          <Button variant="outline" busy={scanning} disabled={!locations?.logFound} onClick={() => locations && open(locations.logPath)}>
            <RefreshCw className="size-4" aria-hidden /> Read again
          </Button>
        }
      />

      {!account && (
        <Callout tone="warning" action={<Button size="sm" onClick={signIn}>Sign in</Button>}>
          Sign in to upload runs. They're tied to your account, so only you can manage and delete them.
        </Callout>
      )}

      <Card
        title="Locations"
        icon={<FolderOpen className="size-4" aria-hidden />}
        action={<Button size="sm" variant="outline" onClick={useGameFolder}>Find from game folder</Button>}
      >
        {locations && (
          <div>
            <LocationRow label="Combat log" path={locations.logPath} found={locations.logFound} onChange={() => change("log")} />
            <LocationRow
              label="WoW Logs AS save"
              path={locations.addonPath}
              found={locations.addonFound}
              chosen={locations.addonChosen}
              onChange={() => change("addon")}
              onReset={() => reset("addon")}
            />
            <LocationRow
              label="Azeroth Stats save (by Maidenhead)"
              path={locations.statsPath}
              found={locations.statsFound}
              chosen={locations.statsChosen}
              onChange={() => change("stats")}
              onReset={() => reset("stats")}
            />
            <LocationRow label="Clean copies folder" path={locations.copiesDir} folder onChange={() => change("copies")} />
          </div>
        )}
        <p className="mt-2 text-xs text-muted-foreground">
          These files are only read. Your log and both saves are never changed.
        </p>
      </Card>

      {!scan && !scanning && (
        <EmptyState title="No log found">
          Use <span className="font-medium">Find from game folder</span> and choose the folder with{" "}
          <span className="font-mono">AzerothSurvivors.exe</span>, or change the combat log location above.
        </EmptyState>
      )}

      {scan?.addonError && <Callout tone="warning">{scan.addonError}</Callout>}
      {scan?.statsError && <Callout tone="warning">{scan.statsError}</Callout>}

      {scan && (
        <>
          <Card
            title={`${runs.length} ${runs.length === 1 ? "run" : "runs"} in ${fileName(scan.path)}`}
            action={
              <div className="flex gap-1">
                {uploadedCount > 0 && (
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => selectWhere((run) => run.activeMs >= scan.minActiveMs && !known.has(run.index))}
                  >
                    New only
                  </Button>
                )}
                <Button size="sm" variant="ghost" onClick={() => selectWhere((run) => run.activeMs >= scan.minActiveMs)}>All</Button>
                {withAddon > 0 && (
                  <Button size="sm" variant="ghost" onClick={() => selectWhere((run) => !!matches.get(run.index)?.addon)}>With addon data</Button>
                )}
                <Button size="sm" variant="ghost" onClick={() => setSelected(new Set())}>None</Button>
              </div>
            }
          >
            <p className="mb-3 text-xs text-muted-foreground">
              {uploadedCount > 0 &&
                `${uploadedCount} already uploaded, so ${uploadedCount === 1 ? "it starts" : "they start"} unticked. `}
              {withAddon} of {runs.length} with WoW Logs AS data{scan.addonRuns > 0 && ` (${scan.addonRuns} runs in the save)`}.{" "}
              {withStats} with Azeroth Stats data{scan.statsRuns > 0 && ` (${scan.statsRuns} runs in the save)`}.
            </p>
            <div className="max-h-96 space-y-2 overflow-y-auto pr-1">
              {runs.map((run) => (
                <RunRow
                  key={run.index}
                  run={run}
                  match={matches.get(run.index)}
                  checked={selected.has(run.index)}
                  short={run.activeMs < scan.minActiveMs}
                  uploadedUrl={known.has(run.index) ? `${siteUrl}/as/${known.get(run.index)}` : undefined}
                  onToggle={() => toggle(run.index)}
                />
              ))}
            </div>
          </Card>

          <Card title={`${chosen.length} ${chosen.length === 1 ? "run" : "runs"} chosen`}>
            <div className="space-y-3">
              <SwitchRow
                checked={sendAddon}
                onChange={setSendAddon}
                disabled={chosenAddon === 0}
                label="Include WoW Logs AS data"
                note={`${chosenAddon} of ${chosen.length} chosen runs have it`}
              />
              <SwitchRow
                checked={sendStats}
                onChange={setSendStats}
                disabled={chosenStats === 0}
                label="Include Azeroth Stats (by Maidenhead)"
                note={`${chosenStats} of ${chosen.length} chosen runs have it`}
              />
              <p className="text-xs text-muted-foreground">
                Only the chosen runs' addon data is used. The rest of each save stays on this computer.
              </p>
            </div>
          </Card>

          <Card title="Upload" icon={<UploadCloud className="size-4" aria-hidden />}>
            <div className="space-y-3">
              <Input
                aria-label="Hero name"
                maxLength={32}
                placeholder="Hero name (optional: the log doesn't name your hero)"
                value={heroName}
                onChange={(event) => setHeroName(event.target.value)}
              />
              <SwitchRow checked={isPrivate} onChange={setIsPrivate} label="Private" note="Only you can see these runs." />
              <p className="text-xs text-muted-foreground">
                You can keep up to 300 private runs and delete any run at any time. Deleted runs can stay in our backups for up to 3 days.
              </p>
              <div className="flex flex-wrap items-center justify-end gap-3">
                {uploading && (
                  <span className={`text-xs ${waitedLong ? "text-gold" : "text-muted-foreground"}`}>
                    {status?.status === "PROCESSING"
                      ? "Building your reports..."
                      : waitedLong
                        ? "Still waiting for the server to start on it. This is taking longer than usual."
                        : "Waiting in the queue..."}
                  </span>
                )}
                <Button variant="primary" busy={uploading} disabled={!account || chosen.length === 0} onClick={upload}>
                  <UploadCloud className="size-4" aria-hidden />
                  Upload {chosen.length} {chosen.length === 1 ? "run" : "runs"}
                </Button>
              </div>
            </div>
          </Card>

          <Card title="Clean copy" icon={<Copy className="size-4" aria-hidden />}>
            <div className="space-y-3">
              <p className="text-sm text-muted-foreground">
                Writes the chosen runs to a new folder, with only their WoW Logs AS and Azeroth Stats data, so the
                three files match. Nothing existing is changed. The copy uploads here or on the website like the originals.
              </p>
              {copy && (
                <Callout
                  tone="info"
                  action={<Button size="sm" variant="outline" onClick={() => OpenAsFolder(copy.dir).catch((error) => toast.error(errorText(error)))}>Open folder</Button>}
                >
                  <span className="block">
                    Saved {copy.runs} {copy.runs === 1 ? "run" : "runs"}
                    {copy.addonRuns > 0 && `, WoW Logs AS data for ${copy.addonRuns}`}
                    {copy.statsRuns > 0 && `, Azeroth Stats data for ${copy.statsRuns}`}.
                  </span>
                  <span className="block truncate font-mono text-xs" title={copy.dir}>{fileName(copy.dir)}</span>
                </Callout>
              )}
              <div className="flex justify-end">
                <Button variant="outline" busy={copying} disabled={chosen.length === 0 || !locations?.copiesDir} onClick={createCopy}>
                  <Copy className="size-4" aria-hidden /> Create clean copy
                </Button>
              </div>
            </div>
          </Card>
        </>
      )}

      {finished && status && (
        <Callout tone={status.status === "FAILED" ? "warning" : "info"}>
          <span className="flex items-center gap-2">
            {status.status === "DONE" && <CheckCircle2 className="size-4" aria-hidden />}
            {status.status === "FAILED"
              ? "The upload failed. Please try again."
              : `${status.runsCreated} added, ${status.runsDuplicate} already uploaded.`}
            {status.runsSkippedQuota > 0 && ` ${status.runsSkippedQuota} not saved: you've reached your limit of private runs.`}
            {status.status === "DONE" && (
              <button type="button" className="cursor-pointer underline" onClick={() => openOnSite(`${siteUrl}/as`)}>
                View runs
              </button>
            )}
          </span>
        </Callout>
      )}
    </div>
  );
}
