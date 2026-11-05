import { useMemo, useState } from "react";
import { toast } from "sonner";
import { CheckCircle2, Clock, Download, GitMerge, Plus, Trash2, UploadCloud, Zap } from "lucide-react";
import { MergeAndUpload, SaveMergedFile, ScanMergeFiles, SelectMergeFiles } from "../../wailsjs/go/main/App";
import { main } from "../../wailsjs/go/models";
import { Badge, Button, Callout, Card, EmptyState, PageHeader, Stat, Switch, Table } from "../ui";
import { useApp } from "../state/AppContext";
import { formatBytes, formatDuration, formatLogTime } from "../state/format";

function linkLabel(link: main.MergeLink): { text: string; tone: "warning" | "brand" | "neutral" } {
  if (link.overlap) return { text: `${formatDuration(-link.gapMs)} overlap`, tone: "warning" };
  if (link.nextDay) return { text: `Next session, ${formatDuration(link.gapMs)} later`, tone: "brand" };
  return { text: `${formatDuration(link.gapMs)} gap`, tone: "neutral" };
}

function Timeline({ plan }: { plan: main.MergePlan }) {
  const files = plan.files ?? [];
  return (
    <div className="flex flex-wrap items-stretch gap-2">
      {files.map((file, index) => {
        const link = plan.links?.[index];
        const label = link ? linkLabel(link) : null;
        return (
          <div key={file.path} className="contents">
            <div className="min-w-40 flex-1 rounded-control border border-brand-text/40 bg-accent/40 px-3 py-2">
              <p className="truncate text-sm font-medium" title={file.name}>{file.name}</p>
              <p className="font-mono text-xs text-brand-text">
                {file.error ? "unreadable" : `${formatDuration(file.lastMs - file.firstMs)} · ${file.bosses?.length ?? 0} bosses`}
              </p>
            </div>
            {label && (
              <div className="flex items-center">
                <Badge tone={label.tone}>
                  {link?.overlap ? <Zap className="size-3" aria-hidden /> : <Clock className="size-3" aria-hidden />}
                  {label.text}
                </Badge>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

export function MergeScreen() {
  const { server, canMarkPrivate, monitorUpload, setInstanceFlow, setScreen, setUploadProgress } = useApp();
  const [paths, setPaths] = useState<string[]>([]);
  const [plan, setPlan] = useState<main.MergePlan | null>(null);
  const [scanning, setScanning] = useState(false);
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [unlisted, setUnlisted] = useState(false);

  function scan(next: string[]) {
    setPaths(next);
    if (next.length === 0) {
      setPlan(null);
      return;
    }
    setScanning(true);
    ScanMergeFiles(next)
      .then(setPlan)
      .catch((err: unknown) => toast.error("Could not read the files", { description: String(err) }))
      .finally(() => setScanning(false));
  }

  function addFiles() {
    SelectMergeFiles()
      .then((picked) => {
        if (picked?.length) scan([...new Set([...paths, ...picked])]);
      })
      .catch((err: unknown) => toast.error("Could not open files", { description: String(err) }));
  }

  function save() {
    setSaving(true);
    SaveMergedFile(paths)
      .then((path) => {
        if (path) toast.success("Merged log saved", { description: path });
      })
      .catch((err: unknown) => toast.error("Could not save", { description: String(err) }))
      .finally(() => setSaving(false));
  }

  function upload() {
    setUploading(true);
    const wantPrivate = unlisted && canMarkPrivate;
    MergeAndUpload(paths, server, wantPrivate)
      .then((response) => {
        if (response.autoQueued) {
          toast.success("Merged raid queued for processing");
          monitorUpload(response.preprocessId, false);
          scan([]);
          setScreen("upload");
        } else {
          setInstanceFlow({
            preprocessId: response.preprocessId,
            instances: (response.instances ?? []).map((instance) => ({ ...instance, unlisted: wantPrivate })),
            hasMultipleDetectedServers: !!response.hasMultipleDetectedServers,
            trackLiveLog: false,
          });
        }
      })
      .catch((err: unknown) => toast.error("Merge failed", { description: String(err), duration: 15000 }))
      .finally(() => {
        setUploading(false);
        setUploadProgress(null);
      });
  }

  const files = plan?.files ?? [];
  const overlapping = useMemo(() => (plan?.links ?? []).some((link) => link.overlap), [plan]);

  return (
    <div className="space-y-4">
      <PageHeader
        title="Merge split log files"
        description="Combine a raid split by a disconnect, or one lockout raided over several days, into a single report before uploading."
        action={(
          <Button variant="outline" onClick={addFiles} busy={scanning}>
            <Plus className="size-4" aria-hidden /> Add log files…
          </Button>
        )}
      />

      {!plan ? (
        <EmptyState title={scanning ? "Reading files…" : "Add two or more log files"}>
          Files are put in time order automatically. Nothing is uploaded until you confirm.
        </EmptyState>
      ) : (
        <>
          <Card
            title="Raid timeline"
            icon={<GitMerge className="size-4 text-brand-text" aria-hidden />}
            action={<span className="font-mono text-xs text-muted-foreground">Active time {formatDuration(plan.activeMs)}</span>}
          >
            <Timeline plan={plan} />
            {overlapping && (
              <p className="mt-3 text-xs text-muted-foreground">Lines recorded in both files are removed during the merge.</p>
            )}
          </Card>

          {plan.warnings?.map((warning) => <Callout key={warning} tone="warning">{warning}</Callout>)}
          {plan.multiDay && (
            <Callout>Multi-day raids are ranked per boss but excluded from full-clear speed rankings.</Callout>
          )}

          <Card
            title="Merged raid"
            icon={<CheckCircle2 className="size-4 text-success" aria-hidden />}
            action={<span className="font-mono text-xs text-success">{plan.bosses?.length ?? 0} bosses fought</span>}
          >
            <div className="flex flex-wrap gap-8">
              <Stat label="Files" value={files.length} />
              <Stat label="Combined size" value={formatBytes(plan.totalBytes)} />
              <Stat label="Time span" value={formatDuration(plan.spanMs)} />
            </div>
            {!!plan.bosses?.length && (
              <div className="mt-3 flex flex-wrap gap-1">
                {plan.bosses.map((boss) => <Badge key={boss}>{boss}</Badge>)}
              </div>
            )}
          </Card>

          <Table columns={["Order", "File", "Time range (log time)", "Size", "Bosses", ""]}>
            {files.map((file, index) => (
              <tr key={file.path}>
                <td className="font-mono text-xs">#{index + 1}</td>
                <td className="max-w-56">
                  <p className="truncate font-mono text-xs font-medium" title={file.path}>{file.name}</p>
                  {file.error && <p className="text-xs text-destructive">{file.error}</p>}
                </td>
                <td className="font-mono text-xs whitespace-nowrap">
                  {file.error ? "—" : `${formatLogTime(file.firstMs)} → ${formatLogTime(file.lastMs)}`}
                </td>
                <td className="font-mono text-xs whitespace-nowrap">{formatBytes(file.sizeBytes)}</td>
                <td className="text-xs">{file.bosses?.join(", ") || "—"}</td>
                <td>
                  <button
                    type="button"
                    className="rounded p-1 text-destructive hover:bg-muted"
                    aria-label={`Remove ${file.name}`}
                    onClick={() => scan(paths.filter((path) => path !== file.path))}
                  >
                    <Trash2 className="size-4" aria-hidden />
                  </button>
                </td>
              </tr>
            ))}
          </Table>

          <Card>
            <div className="flex flex-wrap items-center justify-between gap-3">
              <label className="flex items-center gap-2 text-sm">
                <Switch label="Private" checked={unlisted && canMarkPrivate} disabled={!canMarkPrivate} onChange={setUnlisted} />
                Private (unlisted)
              </label>
              <div className="flex flex-wrap gap-2">
                <Button variant="outline" busy={saving} disabled={!plan.canMerge} onClick={save}>
                  <Download className="size-4" aria-hidden /> Save merged file
                </Button>
                <Button variant="primary" busy={uploading} disabled={!plan.canMerge || !server} onClick={upload}>
                  <UploadCloud className="size-4" aria-hidden /> Merge &amp; upload
                </Button>
              </div>
            </div>
            {!server && <p className="mt-2 text-xs text-muted-foreground">Choose a server at the top first.</p>}
          </Card>
        </>
      )}
    </div>
  );
}
