import { ExternalLink, Moon, Server, Sun } from "lucide-react";
import { OpenLogPage } from "../../wailsjs/go/main/App";
import { Badge, Select } from "../ui";
import { useApp } from "../state/AppContext";
import { formatTimeAgo, watcherBadge } from "../state/format";

export function StatusStrip() {
  const { servers, server, setServer, season, watcher, history, theme, setThemeMode } = useApp();
  const badge = watcherBadge(watcher?.status, watcher?.running);
  const latest = history.find((entry) => entry.status === "uploaded" && entry.logIds?.length);

  return (
    <div className="flex h-12 shrink-0 items-center gap-3 border-b border-border bg-card px-4">
      <Server className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      <Select aria-label="Server" className="w-52 shrink-0" value={server} onChange={(event) => setServer(event.target.value)}>
        <option value="">Choose a server…</option>
        {servers.map((option) => (
          <option key={option.value} value={option.value}>{option.label}</option>
        ))}
      </Select>
      {season != null && <Badge>Season {season}</Badge>}
      <Badge tone={badge.tone}>{badge.label}</Badge>

      <div className="min-w-0 flex-1" />

      {latest && (
        <button
          type="button"
          className="flex min-w-0 items-center gap-1 truncate font-mono text-xs text-brand-text hover:underline"
          onClick={() => void OpenLogPage(latest.logIds[0])}
        >
          <span className="truncate">{latest.raids?.[0] || "Latest log"} · {formatTimeAgo(latest.startedAt)}</span>
          <ExternalLink className="size-3 shrink-0" aria-hidden />
        </button>
      )}
      <button
        type="button"
        aria-label={theme === "dark" ? "Switch to light theme" : "Switch to dark theme"}
        className="shrink-0 rounded-control p-2 text-muted-foreground hover:bg-muted"
        onClick={() => setThemeMode(theme === "dark" ? "light" : "dark")}
      >
        {theme === "dark" ? <Sun className="size-4" aria-hidden /> : <Moon className="size-4" aria-hidden />}
      </button>
    </div>
  );
}
