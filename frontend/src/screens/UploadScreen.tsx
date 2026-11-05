import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import { Archive, Check, FileText, FolderOpen, Lock, UploadCloud, ExternalLink } from "lucide-react";
import {
  ArchiveCombatLog,
  GetUploadPreview,
  OpenAllLogsPage,
  SelectCombatLogFile,
  UploadLog,
} from "../../wailsjs/go/main/App";
import { main } from "../../wailsjs/go/models";
import { Badge, Button, Callout, Card, PageHeader, PremiumBadge, Stat, Switch } from "../ui";
import { useApp } from "../state/AppContext";
import { formatBytes, formatDateTime } from "../state/format";
import { HistoryTable } from "./HistoryTable";

const STAGES = [
  { key: "compressing", label: "Compressing" },
  { key: "uploading", label: "Uploading" },
  { key: "detecting", label: "Finding raids" },
  { key: "queued", label: "Queued" },
  { key: "done", label: "Processing" },
];

function ProgressPanel() {
  const { uploadProgress, jobProgress } = useApp();
  const stage = jobProgress ? (jobProgress.queuePosition ? "queued" : "done") : uploadProgress?.stage;
  if (!stage) return null;
  const activeIndex = STAGES.findIndex((item) => item.key === stage);
  const percent = !jobProgress && uploadProgress ? uploadProgress.percent : null;

  return (
    <Card title="Upload progress" icon={<UploadCloud className="size-4 text-brand-text" aria-hidden />}>
      <ol className="flex flex-wrap gap-x-4 gap-y-1 text-xs">
        {STAGES.map((item, index) => (
          <li key={item.key} className={`flex items-center gap-1 ${index === activeIndex ? "font-semibold text-foreground" : index < activeIndex ? "text-success" : "text-muted-foreground"}`}>
            {index < activeIndex && <Check className="size-3" aria-hidden />}
            {item.label}
          </li>
        ))}
      </ol>
      {percent != null && (
        <div className="mt-3">
          <div className="h-2 overflow-hidden rounded-full bg-muted">
            <div className="h-full rounded-full bg-brand transition-[width]" style={{ width: `${percent}%` }} />
          </div>
          <p className="mt-1 font-mono text-xs text-muted-foreground">
            {percent}%{uploadProgress && uploadProgress.bytesTotal > 0 ? ` · ${formatBytes(uploadProgress.bytesDone)} of ${formatBytes(uploadProgress.bytesTotal)}` : ""}
          </p>
        </div>
      )}
      {jobProgress?.queuePosition ? (
        <p className="mt-2 text-sm text-muted-foreground">Position {jobProgress.queuePosition} in the server queue.</p>
      ) : jobProgress ? (
        <p className="mt-2 text-sm text-muted-foreground">
          {jobProgress.jobsCompleted} of {jobProgress.totalJobs} raids processed.
        </p>
      ) : null}
    </Card>
  );
}

export function UploadScreen() {
  const {
    logDirectory, chooseLogDirectory, server, canMarkPrivate, isPremium, history, setInstanceFlow,
    monitorUpload, uploadProgress, setUploadProgress, jobProgress, refreshHistory,
  } = useApp();
  const [customFile, setCustomFile] = useState("");
  const [preview, setPreview] = useState<main.UploadPreview | null>(null);
  const [newOnly, setNewOnly] = useState(true);
  const [unlisted, setUnlisted] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [archiving, setArchiving] = useState(false);

  const refreshPreview = useCallback(() => {
    if (!logDirectory && !customFile) {
      setPreview(null);
      return;
    }
    GetUploadPreview(customFile, logDirectory).then(setPreview).catch(console.error);
  }, [customFile, logDirectory]);

  useEffect(() => {
    refreshPreview();
    const timer = window.setInterval(refreshPreview, 15000);
    return () => window.clearInterval(timer);
  }, [refreshPreview]);

  useEffect(() => {
    refreshHistory();
  }, [refreshHistory]);

  function browse() {
    SelectCombatLogFile()
      .then((path) => {
        if (path) setCustomFile(path);
      })
      .catch((err: unknown) => toast.error("Could not open the file", { description: String(err) }));
  }

  const hasDelta = !customFile && !!preview?.hasPrevious;
  const useNewOnly = hasDelta && newOnly;
  const uploadBytes = preview ? (useNewOnly ? preview.newBytes : preview.sizeBytes) : 0;
  const tooLarge = uploadBytes > 1024 * 1024 * 1024;
  const nothingNew = useNewOnly && preview?.newBytes === 0;
  const canUpload = !!preview?.exists && !!server && !uploading && !nothingNew && !tooLarge;
  const trackLiveLog = !customFile;

  function upload() {
    setUploading(true);
    setUploadProgress({ stage: "compressing", percent: 0, bytesDone: 0, bytesTotal: uploadBytes });
    const wantPrivate = unlisted && canMarkPrivate;
    UploadLog(new main.UploadRequest({
      path: customFile,
      logDirectory,
      serverName: server,
      unlisted: wantPrivate,
      newContentOnly: useNewOnly,
    }))
      .then((response) => {
        if (response.autoQueued) {
          toast.success("Raid queued for processing");
          monitorUpload(response.preprocessId, trackLiveLog);
        } else {
          setInstanceFlow({
            preprocessId: response.preprocessId,
            instances: (response.instances ?? []).map((instance) => ({ ...instance, unlisted: wantPrivate })),
            hasMultipleDetectedServers: !!response.hasMultipleDetectedServers,
            trackLiveLog,
          });
        }
      })
      .catch((err: unknown) => toast.error("Upload failed", { description: String(err), duration: 15000 }))
      .finally(() => {
        setUploading(false);
        setUploadProgress(null);
        refreshPreview();
      });
  }

  function archive() {
    setArchiving(true);
    ArchiveCombatLog(logDirectory)
      .then((path) => toast.success("Combat log archived", { description: path }))
      .catch((err: unknown) => toast.error("Could not archive", { description: String(err) }))
      .finally(() => {
        setArchiving(false);
        refreshPreview();
      });
  }

  const sourcePath = customFile || preview?.path || logDirectory;

  return (
    <div className="space-y-4">
      <PageHeader
        title="Upload combat log"
        description="Send a raid to wow-logs.co.in for boss, speed and parse analysis."
        action={(
          <Button variant="outline" onClick={() => void OpenAllLogsPage()}>
            <ExternalLink className="size-4" aria-hidden /> View all logs
          </Button>
        )}
      />

      {(uploading || uploadProgress || jobProgress) && <ProgressPanel />}

      <Card
        title="Log source"
        icon={<FileText className="size-4 text-brand-text" aria-hidden />}
        action={preview?.exists ? <Badge tone="success">Found</Badge> : preview ? <Badge tone="danger">Not found</Badge> : undefined}
      >
        {!logDirectory && !customFile ? (
          <Callout action={<Button size="sm" variant="primary" onClick={() => void chooseLogDirectory()}>Choose folder</Button>}>
            Choose your WoW <span className="font-mono">Logs</span> folder to get started.
          </Callout>
        ) : (
          <>
            <div className="flex gap-2">
              <div className="flex h-9 min-w-0 flex-1 items-center rounded-control border border-border bg-input px-3 font-mono text-xs" title={sourcePath}>
                <span className="truncate">{sourcePath}</span>
              </div>
              {customFile && <Button variant="ghost" onClick={() => setCustomFile("")}>Use live log</Button>}
              <Button variant="outline" onClick={browse}>
                <FolderOpen className="size-4" aria-hidden /> Browse…
              </Button>
            </div>
            {preview?.exists && (
              <div className="mt-3 flex flex-wrap items-end justify-between gap-4">
                <div className="flex flex-wrap gap-8">
                  <Stat label="File size" value={formatBytes(preview.sizeBytes)} />
                  <Stat label="Last modified" value={formatDateTime(preview.modifiedAt)} />
                  {hasDelta && <Stat label="New since last upload" value={formatBytes(preview.newBytes)} />}
                </div>
                {!customFile && (
                  <Button
                    variant="ghost"
                    size="sm"
                    busy={archiving}
                    disabled={preview.wowRunning}
                    title={preview.wowRunning ? "Close WoW first" : "Move this file to Logs/archive"}
                    onClick={archive}
                  >
                    <Archive className="size-3.5" aria-hidden /> Archive &amp; clear
                  </Button>
                )}
              </div>
            )}
          </>
        )}
      </Card>

      {tooLarge && (
        <Callout tone="warning">
          This is over the 1024 MB upload limit. Close WoW and use Archive &amp; clear, or upload only new content.
        </Callout>
      )}

      <div className="grid gap-4 md:grid-cols-2">
        <Card title="What to upload">
          <label className="flex items-start justify-between gap-3">
            <span>
              <span className="block text-sm font-medium">Only new content</span>
              <span className="block text-xs text-muted-foreground">
                {hasDelta ? "Skips everything already uploaded from this log." : "Available after your first upload from this log."}
              </span>
            </span>
            <Switch label="Only new content" checked={useNewOnly} disabled={!hasDelta} onChange={setNewOnly} />
          </label>
        </Card>
        <Card title="Visibility" action={<PremiumBadge />}>
          <label className="flex items-start justify-between gap-3">
            <span>
              <span className="flex items-center gap-1.5 text-sm font-medium">
                <Lock className="size-3.5 text-gold" aria-hidden /> Private (unlisted)
              </span>
              <span className="block text-xs text-muted-foreground">
                {canMarkPrivate
                  ? `Not ranked or listed.${isPremium ? "" : " Limited on the free plan."} Auto-uploads are always public.`
                  : "Sign in with a personal account to upload private logs."}
              </span>
            </span>
            <Switch label="Private" checked={unlisted && canMarkPrivate} disabled={!canMarkPrivate} onChange={setUnlisted} />
          </label>
        </Card>
      </div>

      <Card>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="min-w-0 text-sm">
            <p className="font-medium">{nothingNew ? "Nothing new since your last upload" : server ? "Ready to upload" : "Choose a server at the top first"}</p>
            {preview?.exists && !nothingNew && <p className="text-xs text-muted-foreground">{formatBytes(uploadBytes)} will be compressed and sent.</p>}
          </div>
          <Button variant="primary" disabled={!canUpload} busy={uploading} onClick={upload}>
            <UploadCloud className="size-4" aria-hidden /> Upload &amp; process
          </Button>
        </div>
      </Card>

      <div>
        <h2 className="mb-2 text-sm font-semibold">Recent uploads</h2>
        <HistoryTable
          entries={history.slice(0, 8)}
          emptyTitle="No uploads yet"
          emptyText="Your uploads from this computer will appear here."
        />
      </div>
    </div>
  );
}
