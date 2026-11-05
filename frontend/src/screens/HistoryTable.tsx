import { useState } from "react";
import { toast } from "sonner";
import { ExternalLink, Lock } from "lucide-react";
import { OpenLogPage, UploadLog } from "../../wailsjs/go/main/App";
import { main } from "../../wailsjs/go/models";
import { Badge, Button, EmptyState, Table } from "../ui";
import { useApp } from "../state/AppContext";
import { formatDateTime, formatDuration, historyBadge } from "../state/format";

export function HistoryTable({ entries, emptyTitle, emptyText }: {
  entries: main.HistoryEntry[];
  emptyTitle: string;
  emptyText: string;
}) {
  const { server, monitorUpload, setInstanceFlow, setUploadProgress } = useApp();
  const [retrying, setRetrying] = useState<string | null>(null);

  if (entries.length === 0) {
    return <EmptyState title={emptyTitle}>{emptyText}</EmptyState>;
  }

  function retry(entry: main.HistoryEntry) {
    if (!entry.retryPath) return;
    setRetrying(entry.retryPath);
    UploadLog(new main.UploadRequest({
      path: entry.retryPath,
      serverName: entry.serverName || server,
      unlisted: false,
      newContentOnly: false,
    }))
      .then((response) => {
        if (response.autoQueued) {
          monitorUpload(response.preprocessId, false);
        } else {
          setInstanceFlow({
            preprocessId: response.preprocessId,
            instances: response.instances ?? [],
            hasMultipleDetectedServers: !!response.hasMultipleDetectedServers,
            trackLiveLog: false,
          });
        }
      })
      .catch((err: unknown) => toast.error("Retry failed", { description: String(err) }))
      .finally(() => {
        setRetrying(null);
        setUploadProgress(null);
      });
  }

  return (
    <Table columns={["Raid", "Bosses", "Started", "Duration", "Status", "Log"]}>
      {entries.map((entry, index) => {
        const badge = historyBadge(entry.status);
        const raid = entry.raids?.length ? entry.raids.join(", ") : entry.message || "Upload";
        return (
          <tr key={`${entry.preprocessId}-${index}`}>
            <td className="max-w-64">
              <span className="flex items-center gap-1.5 font-medium">
                <span className="truncate" title={raid}>{raid}</span>
                {entry.unlisted && <Lock className="size-3.5 shrink-0 text-gold" aria-label="Private" />}
              </span>
            </td>
            <td className="font-mono text-xs">{entry.bosses?.length || "—"}</td>
            <td className="whitespace-nowrap">{formatDateTime(entry.startedAt)}</td>
            <td className="font-mono text-xs whitespace-nowrap">{entry.durationMs ? formatDuration(entry.durationMs) : "—"}</td>
            <td title={entry.message || undefined}><Badge tone={badge.tone}>{badge.label}</Badge></td>
            <td className="whitespace-nowrap">
              <span className="flex items-center gap-2">
                {entry.logIds?.map((id) => (
                  <button key={id} type="button" className="inline-flex items-center gap-1 font-mono text-xs text-brand-text hover:underline" onClick={() => void OpenLogPage(id)}>
                    #{id} <ExternalLink className="size-3" aria-hidden />
                  </button>
                ))}
                {entry.retryPath && (
                  <Button size="sm" variant="outline" busy={retrying === entry.retryPath} onClick={() => retry(entry)}>Retry</Button>
                )}
                {!entry.logIds?.length && !entry.retryPath && <span className="text-muted-foreground">—</span>}
              </span>
            </td>
          </tr>
        );
      })}
    </Table>
  );
}
