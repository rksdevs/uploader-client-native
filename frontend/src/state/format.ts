import type { Tone } from "../ui";

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

export function formatDuration(ms: number): string {
  const totalMinutes = Math.round(ms / 60000);
  if (totalMinutes < 1) return `${Math.max(0, Math.round(ms / 1000))}s`;
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  if (hours >= 24) return `${Math.floor(hours / 24)}d ${hours % 24}h`;
  return hours > 0 ? `${hours}h ${minutes}m` : `${minutes}m`;
}

export function formatDateTime(value: string | undefined): string {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleString(undefined, { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
}

export function formatTimeAgo(value: string | undefined): string {
  if (!value) return "";
  const ms = Date.now() - new Date(value).getTime();
  if (Number.isNaN(ms)) return "";
  if (ms < 60000) return "just now";
  return `${formatDuration(ms)} ago`;
}

export function formatLogTime(ms: number): string {
  return new Date(ms).toLocaleString(undefined, {
    timeZone: "UTC",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

export function historyBadge(status: string): { tone: Tone; label: string } {
  switch (status) {
    case "uploaded":
      return { tone: "success", label: "Done" };
    case "processing":
      return { tone: "brand", label: "Processing" };
    case "partial":
      return { tone: "warning", label: "Partial" };
    case "failed":
      return { tone: "danger", label: "Failed" };
    case "timed_out":
      return { tone: "warning", label: "Still queued" };
    case "skipped":
      return { tone: "neutral", label: "Skipped" };
    default:
      return { tone: "neutral", label: status };
  }
}

export function watcherBadge(status: string | undefined, running: boolean | undefined): { tone: Tone; label: string } {
  if (!running) return { tone: "neutral", label: "Auto-upload off" };
  switch (status) {
    case "watching":
      return { tone: "success", label: "Watching" };
    case "waiting_wow":
      return { tone: "brand", label: "WoW running, uploads when it closes" };
    case "waiting_stable":
      return { tone: "brand", label: "Waiting for log to settle" };
    case "staging_ready":
    case "uploading":
      return { tone: "brand", label: "Uploading" };
    case "awaiting_selection":
    case "awaiting_server_drift":
      return { tone: "warning", label: "Needs your input" };
    case "paused":
      return { tone: "neutral", label: "Paused" };
    case "error":
      return { tone: "danger", label: "Error" };
    default:
      return { tone: "neutral", label: status || "Unknown" };
  }
}

export function watcherDetail(status: string | undefined, detail: string | undefined): string {
  if (detail) return detail;
  return watcherBadge(status, true).label;
}
