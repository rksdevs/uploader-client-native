import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import { Copy, Radio, SquareTerminal } from "lucide-react";
import {
  DisableAutoUpload,
  EstablishAutoUploadBaseline,
  GetAutoUploadSettings,
  GetRecentLogLines,
  SetMinimizeToTray,
} from "../../wailsjs/go/main/App";
import { main } from "../../wailsjs/go/models";
import { Badge, Button, Callout, Card, PageHeader, PremiumBadge, Stat, Switch } from "../ui";
import { useApp } from "../state/AppContext";
import { formatDateTime, watcherBadge, watcherDetail } from "../state/format";
import { HistoryTable } from "./HistoryTable";

export function AutoUploadScreen() {
  const { server, logDirectory, account, isPremium, signIn, watcher, refreshWatcher, history, servers } = useApp();
  const [settings, setSettings] = useState<main.AutoUploadSettingsResponse | null>(null);
  const [lines, setLines] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);

  const refresh = useCallback(() => {
    GetAutoUploadSettings().then(setSettings).catch(console.error);
    refreshWatcher();
    GetRecentLogLines(logDirectory, 8).then((recent) => setLines(recent ?? [])).catch(console.error);
  }, [logDirectory, refreshWatcher]);

  useEffect(() => {
    refresh();
    const timer = window.setInterval(refresh, 5000);
    return () => window.clearInterval(timer);
  }, [refresh, server, account]);

  function toggle(next: boolean) {
    setBusy(true);
    const action = next
      ? EstablishAutoUploadBaseline(server).then((preview) => toast.success("Auto-upload on", { description: preview.message }))
      : DisableAutoUpload().then(() => toast.info("Auto-upload off"));
    action
      .catch((err: unknown) => toast.error(next ? "Could not turn on auto-upload" : "Could not turn off auto-upload", { description: String(err) }))
      .finally(() => {
        setBusy(false);
        refresh();
      });
  }

  function setTray(enabled: boolean) {
    SetMinimizeToTray(enabled).then(refresh).catch((err: unknown) => toast.error("Could not save", { description: String(err) }));
  }

  function copyDevice() {
    void navigator.clipboard.writeText(settings?.deviceId ?? "");
    toast.success("Device ID copied");
  }

  const enabled = !!settings?.enabled;
  const badge = watcherBadge(watcher?.status, watcher?.running);
  const realm = servers.find((option) => option.value === server)?.label || "No server chosen";
  const canToggle = enabled || (!!settings?.canEnable && !!server);

  return (
    <div className="space-y-4">
      <PageHeader
        title="Auto-upload"
        badge={<PremiumBadge />}
        description="Watches your combat log while you raid and uploads new content after WoW closes."
      />

      {!account?.signedIn ? (
        <Callout action={<Button size="sm" variant="primary" onClick={signIn}>Sign in</Button>}>
          Auto-upload needs a Premium account. Sign in to use it.
        </Callout>
      ) : !isPremium ? (
        <Callout tone="warning">Auto-upload needs Premium. Manual upload stays free.</Callout>
      ) : null}
      {settings?.blockReason && !enabled && <Callout tone="warning">{settings.blockReason}</Callout>}

      <Card>
        <div className="flex items-center justify-between gap-4">
          <div className="min-w-0">
            <p className="flex flex-wrap items-center gap-2 font-medium">
              <Radio className="size-4 text-brand-text" aria-hidden /> Background watcher <Badge tone={badge.tone}>{badge.label}</Badge>
            </p>
            <p className="mt-0.5 text-sm text-muted-foreground">
              {enabled ? watcherDetail(watcher?.status, watcher?.detail) : "New raids will not upload until this is on."}
            </p>
          </div>
          <Switch label="Auto-upload" checked={enabled} disabled={busy || !canToggle} onChange={toggle} />
        </div>
      </Card>

      <div className="grid gap-4 md:grid-cols-3">
        <Card><Stat label="Server" value={realm} /></Card>
        <Card>
          <Stat label="Tailing since" value={formatDateTime(settings?.baselineEstablishedAt) || "—"} />
        </Card>
        <Card><Stat label="Watched folder" value={<span title={logDirectory}>{logDirectory || "—"}</span>} /></Card>
      </div>

      <div>
        <h2 className="mb-2 flex items-center gap-2 text-sm font-semibold">
          <SquareTerminal className="size-4 text-brand-text" aria-hidden /> Latest combat log lines
        </h2>
        <div className="overflow-x-auto rounded-card border border-border bg-card p-3 font-mono text-xs leading-6">
          {lines.length === 0 ? (
            <p className="text-muted-foreground">No combat log found in the watched folder.</p>
          ) : lines.map((line, index) => <p key={`${index}-${line.slice(0, 24)}`} className="truncate" title={line}>{line}</p>)}
        </div>
      </div>

      <Card title="Options">
        <div className="grid gap-4 md:grid-cols-2">
          <label className="flex items-center justify-between gap-3 text-sm">
            <span>
              <span className="block font-medium">Minimize to tray when closing</span>
              <span className="block text-xs text-muted-foreground">Keeps the watcher running in the background.</span>
            </span>
            <Switch label="Minimize to tray" checked={!!settings?.minimizeToTray} onChange={setTray} />
          </label>
          <div className="flex items-center justify-between gap-3">
            <Stat label="Device ID" value={settings?.deviceId || "—"} />
            <Button size="sm" variant="outline" onClick={copyDevice}><Copy className="size-3.5" aria-hidden /> Copy</Button>
          </div>
        </div>
      </Card>

      <div>
        <h2 className="mb-2 text-sm font-semibold">Automatic upload history</h2>
        <HistoryTable
          entries={history.filter((entry) => entry.source === "auto").slice(0, 10)}
          emptyTitle="No automatic uploads yet"
          emptyText="They will appear here after your next raid."
        />
      </div>
    </div>
  );
}
