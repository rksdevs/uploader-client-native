import React, { useCallback, useEffect, useState } from "react";
import { Crown, Zap, ZapOff } from "lucide-react";
import { toast } from "sonner";
import ServerSelector, { ServerOption } from "./ServerSelector";
import {
  DisableAutoUpload,
  EstablishAutoUploadBaseline,
  GetAutoUploadSettings,
  SaveDefaultServer,
  SetMinimizeToTray,
} from "../../wailsjs/go/main/App";
import { main } from "../../wailsjs/go/models";
import { EventsOn } from "../../wailsjs/runtime";

interface AutoUploadSettingsProps {
  serverOptions: ServerOption[];
  disabled?: boolean;
  apiToken?: string;
  logDirectory?: string;
  onOpenPremiumSettings?: () => void;
}

interface AutoUploadSettingsState {
  enabled: boolean;
  defaultServer: string;
  deviceId: string;
  hasBaseline: boolean;
  baselineEstablishedAt?: string;
  tailFingerprint?: string[];
  logDirectory: string;
  hasApiToken: boolean;
  canEnable: boolean;
  serverAllowed: boolean;
  blockReason?: string;
  watcher?: main.AutoUploadWatcherStatus;
  minimizeToTray?: boolean;
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}

function formatWatcherStatus(w: main.AutoUploadWatcherStatus | undefined): string {
  if (!w?.status) return "Unknown";
  switch (w.status) {
    case "watching":
      return "Watching for new combat log lines";
    case "waiting_wow":
      return w.detail || "Waiting for WoW to close";
    case "waiting_stable":
      return "Waiting for log file to stabilize";
    case "staging_ready":
      return w.detail || "Slice staged — uploading";
    case "uploading":
      return w.detail || "Uploading combat log slice";
    case "awaiting_selection":
      return w.detail || "Choose raid instances to upload";
    case "awaiting_server_drift":
      return w.detail || "Confirm Warmane realm for new log lines";
    case "paused":
      return "Paused";
    case "error":
      return w.detail || "Error";
    default:
      return w.detail || w.status;
  }
}

const AutoUploadSettings: React.FC<AutoUploadSettingsProps> = ({
  serverOptions,
  disabled = false,
  apiToken = "",
  logDirectory = "",
  onOpenPremiumSettings,
}) => {
  const [settings, setSettings] = useState<AutoUploadSettingsState | null>(null);
  const [defaultServer, setDefaultServer] = useState("");
  const [baselineLines, setBaselineLines] = useState<string[]>([]);
  const [isBusy, setIsBusy] = useState(false);

  const refresh = useCallback(() => {
    GetAutoUploadSettings()
      .then((s: AutoUploadSettingsState) => {
        setSettings(s);
        if (s.defaultServer) setDefaultServer(s.defaultServer);
        if (s.tailFingerprint?.length) setBaselineLines(s.tailFingerprint);
      })
      .catch((err: unknown) => {
        console.error("[AutoUpload] failed to load settings:", err);
      });
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh, apiToken, logDirectory]);

  useEffect(() => {
    const cleanupStatus = EventsOn("auto_upload_status", () => {
      refresh();
    });
    const cleanupStaging = EventsOn(
      "auto_upload_staging_ready",
      (data: { bytes?: number }) => {
        const bytes = data?.bytes ?? 0;
        toast.info("Combat log slice staged", {
          description:
            bytes > 0
              ? `${bytes.toLocaleString()} bytes copied — uploading automatically.`
              : "Starting auto-upload.",
        });
        refresh();
      }
    );
    const cleanupSkipped = EventsOn(
      "auto_upload_skipped",
      (data: { message?: string }) => {
        toast.info("Auto-upload skipped", {
          description: data?.message || "No raid bosses in the new log slice.",
        });
        refresh();
      }
    );
    const cleanupError = EventsOn("auto_upload_error", (data: { message?: string }) => {
      toast.error("Auto-upload failed", {
        description: data?.message || "Unknown error",
      });
      refresh();
    });
    const cleanupComplete = EventsOn("auto_upload_complete", () => {
      toast.success("Auto-upload complete", {
        description: "Tail baseline advanced.",
      });
      refresh();
    });
    const cleanupPremiumLost = EventsOn(
      "auto_upload_premium_lost",
      (data: { message?: string }) => {
        toast.error("Auto-upload disabled", {
          description: data?.message || "Premium subscription is no longer active.",
          duration: 15000,
        });
        refresh();
      }
    );
    return () => {
      cleanupStatus();
      cleanupStaging();
      cleanupSkipped();
      cleanupError();
      cleanupComplete();
      cleanupPremiumLost();
    };
  }, [refresh]);

  useEffect(() => {
    if (!settings?.enabled) return undefined;
    const id = window.setInterval(() => refresh(), 5000);
    return () => window.clearInterval(id);
  }, [settings?.enabled, refresh]);

  const handleDefaultServerChange = (value: string) => {
    setDefaultServer(value);
    SaveDefaultServer(value)
      .then(() => refresh())
      .catch((err: unknown) => {
        toast.error("Could not save default server", { description: String(err) });
      });
  };

  const handleEnable = () => {
    if (!defaultServer) {
      toast.error("Select a default server for auto-upload.");
      return;
    }
    setIsBusy(true);
    EstablishAutoUploadBaseline(defaultServer)
      .then((preview: main.BaselinePreview) => {
        setBaselineLines(preview.lines || []);
        toast.success("Auto-upload enabled", { description: preview.message });
        refresh();
      })
      .catch((err: unknown) => {
        toast.error("Could not enable auto-upload", { description: String(err) });
      })
      .finally(() => setIsBusy(false));
  };

  const handleDisable = () => {
    setIsBusy(true);
    DisableAutoUpload()
      .then(() => {
        toast.info("Auto-upload paused.");
        refresh();
      })
      .catch((err: unknown) => {
        toast.error("Could not disable auto-upload", { description: String(err) });
      })
      .finally(() => setIsBusy(false));
  };

  if (!settings) {
    return (
      <div className="redesign-card auto-upload-card">
        <p className="auto-upload-loading">Loading auto-upload settings…</p>
      </div>
    );
  }

  const gilneasSelected = defaultServer === "Whitemane_Gilneas";

  return (
    <div className="redesign-card auto-upload-card">
      <div className="auto-upload-header">
        <Crown size={18} strokeWidth={2} aria-hidden />
        <h3 className="auto-upload-title">Automatic upload (Premium)</h3>
        {settings.enabled ? (
          <span className="premium-strip-active">Active</span>
        ) : null}
      </div>

      <p className="auto-upload-desc">
        Tail your combat log and upload new raid content automatically after WoW closes.
        Manual upload remains unchanged. Not available on Whitemane-Gilneas.
      </p>

      {settings.enabled && settings.watcher ? (
        <>
          <div className="auto-upload-watcher-strip" role="status">
            <span className="auto-upload-watcher-label">Watcher</span>
            <span className="auto-upload-watcher-value">
              {formatWatcherStatus(settings.watcher)}
            </span>
            {settings.watcher.detail && settings.watcher.status !== "watching" ? (
              <span className="auto-upload-watcher-detail">{settings.watcher.detail}</span>
            ) : null}
          </div>

          {settings.watcher.fileActivity ? (
            <div className="auto-upload-activity" role="status">
              <div className="auto-upload-activity-row">
                <span className="auto-upload-activity-label">Combat log file</span>
                <span
                  className={
                    settings.watcher.fileActivity.hasPendingChanges
                      ? "auto-upload-activity-badge auto-upload-activity-badge--changed"
                      : "auto-upload-activity-badge"
                  }
                >
                  {settings.watcher.fileActivity.hasPendingChanges
                    ? "Changed since baseline"
                    : "No new data"}
                </span>
              </div>
              <dl className="auto-upload-activity-grid">
                <div>
                  <dt>File size</dt>
                  <dd>
                    {settings.watcher.fileActivity.fileExists
                      ? formatBytes(settings.watcher.fileActivity.currentSize)
                      : "Not found"}
                    {settings.watcher.fileActivity.hasPendingChanges
                      ? ` (+${formatBytes(settings.watcher.fileActivity.pendingBytes)} pending)`
                      : ""}
                  </dd>
                </div>
                <div>
                  <dt>Last modified</dt>
                  <dd>
                    {settings.watcher.fileActivity.lastModified
                      ? new Date(settings.watcher.fileActivity.lastModified).toLocaleString()
                      : "—"}
                  </dd>
                </div>
                <div>
                  <dt>WoW client</dt>
                  <dd>
                    {settings.watcher.fileActivity.wowRunning
                      ? "Running"
                      : settings.watcher.fileActivity.wowClosedDetail || "Closed"}
                  </dd>
                </div>
                <div>
                  <dt>File stable</dt>
                  <dd>{settings.watcher.fileActivity.fileStable ? "Yes" : "No (still writing)"}</dd>
                </div>
              </dl>
              {settings.watcher.fileActivity.lastLinePreview ? (
                <p className="auto-upload-activity-latest">
                  <span className="auto-upload-activity-label">Latest line</span>
                  <code>{settings.watcher.fileActivity.lastLinePreview}</code>
                </p>
              ) : null}
            </div>
          ) : null}
        </>
      ) : null}

      <div className="form-group">
        <label htmlFor="auto-upload-server">Default server</label>
        <ServerSelector
          selectedValue={defaultServer}
          onSelect={handleDefaultServerChange}
          disabled={disabled || isBusy || settings.enabled}
          serverOptions={serverOptions.filter((s) => s.value !== "Whitemane_Gilneas")}
        />
        {gilneasSelected ? (
          <p className="auto-upload-warning" role="alert">
            Auto-upload is not supported for Whitemane-Gilneas.
          </p>
        ) : null}
      </div>

      {settings.blockReason && !settings.enabled ? (
        <div className="auto-upload-prereq" role="status">
          <p className="auto-upload-warning">{settings.blockReason}</p>
          {!settings.hasApiToken && onOpenPremiumSettings ? (
            <button
              type="button"
              className="btn-surface btn-with-icon auto-upload-prereq-btn"
              onClick={onOpenPremiumSettings}
            >
              <Crown size={16} strokeWidth={2} aria-hidden />
              Open premium settings
            </button>
          ) : null}
        </div>
      ) : null}

      {baselineLines.length > 0 ? (
        <div className="auto-upload-baseline">
          <p className="auto-upload-baseline-label">
            Tailing from these last events
            {settings.baselineEstablishedAt
              ? ` (since ${new Date(settings.baselineEstablishedAt).toLocaleString()})`
              : ""}
            :
          </p>
          <pre className="auto-upload-baseline-lines">
            {baselineLines.slice(-8).join("\n")}
          </pre>
        </div>
      ) : null}

      <div className="auto-upload-actions">
        {!settings.enabled ? (
          <button
            type="button"
            className="btn-gradient-primary btn-with-icon"
            onClick={handleEnable}
            disabled={disabled || isBusy || !settings.canEnable || gilneasSelected}
            title={
              !settings.canEnable
                ? settings.blockReason
                : "Capture current log tail and enable auto-upload"
            }
          >
            <Zap size={18} strokeWidth={2} aria-hidden />
            Enable auto-upload
          </button>
        ) : (
          <button
            type="button"
            className="btn-surface btn-with-icon"
            onClick={handleDisable}
            disabled={disabled || isBusy}
          >
            <ZapOff size={18} strokeWidth={2} aria-hidden />
            Pause auto-upload
          </button>
        )}
      </div>

      {settings.deviceId ? (
        <p className="auto-upload-device-id" title={settings.deviceId}>
          Device ID: {settings.deviceId.slice(0, 8)}…
        </p>
      ) : null}

      <label className="auto-upload-tray-option">
        <input
          type="checkbox"
          checked={settings.minimizeToTray !== false}
          disabled={disabled || isBusy}
          onChange={(e) => {
            SetMinimizeToTray(e.target.checked)
              .then(() => refresh())
              .catch((err: unknown) => {
                toast.error("Could not save tray setting", { description: String(err) });
              });
          }}
        />
        <span>Minimize to system tray when closing the window</span>
      </label>
    </div>
  );
};

export default AutoUploadSettings;
