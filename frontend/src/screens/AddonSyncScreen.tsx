import { useCallback, useEffect, useState } from "react";
import { CheckCircle2, Download, FolderX, HelpCircle, SquareTerminal, Trophy } from "lucide-react";
import { GetAddonStatus } from "../../wailsjs/go/main/App";
import { main } from "../../wailsjs/go/models";
import { BrowserOpenURL } from "../../wailsjs/runtime";
import AddonPathHelpModal from "../components/AddonPathHelpModal";
import { Badge, Button, Callout, Card, EmptyState, PageHeader, Stat, Table } from "../ui";
import { useApp } from "../state/AppContext";
import { formatTimeAgo } from "../state/format";

export function AddonSyncScreen() {
  const { wowDirectory, chooseWowDirectory, setScreen } = useApp();
  const [status, setStatus] = useState<main.AddonStatus | null>(null);
  const [showHelp, setShowHelp] = useState(false);

  const refresh = useCallback(() => {
    GetAddonStatus().then(setStatus).catch(console.error);
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh, wowDirectory]);

  const datasets = status?.datasets ?? [];

  return (
    <div className="space-y-4">
      <PageHeader
        title="Addon sync"
        description="Rankings you send are written to the WoW Logs Addon for use in-game."
        action={(
          <Button variant="primary" onClick={() => setScreen("rankings")}>
            <Trophy className="size-4" aria-hidden /> Choose rankings to send
          </Button>
        )}
      />

      <Callout>
        <SquareTerminal className="mr-1 inline size-4" aria-hidden />
        Type <span className="font-mono font-semibold">/reload</span> in-game to see newly sent rankings.
      </Callout>

      <div className="grid gap-4 md:grid-cols-2">
        <Card
          title="Game folder"
          action={<Button size="sm" variant="ghost" onClick={() => setShowHelp(true)}><HelpCircle className="size-3.5" aria-hidden /> Path help</Button>}
        >
          {status?.folderFound ? (
            <p className="flex items-start gap-2 text-sm">
              <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-success" aria-hidden />
              <span className="min-w-0">
                <span className="block font-medium text-success">Folder found</span>
                <span className="block truncate font-mono text-xs" title={status.wowDirectory}>{status.wowDirectory}</span>
              </span>
            </p>
          ) : (
            <div className="space-y-2">
              <p className="flex items-center gap-2 text-sm text-warning">
                <FolderX className="size-4" aria-hidden /> Interface/AddOns not found{wowDirectory ? " in this folder" : ""}.
              </p>
              <Button size="sm" variant="outline" onClick={() => void chooseWowDirectory()}>Choose WoW folder</Button>
            </div>
          )}
        </Card>
        <Card
          title="WoW Logs Addon"
          action={
            status?.updateAvailable ? <Badge tone="warning">Update available</Badge>
              : status?.addonInstalled ? <Badge tone="success">Installed</Badge>
              : <Badge>Not installed</Badge>
          }
        >
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex gap-8">
              <Stat label="Installed" value={status?.installedVersion || "—"} />
              <Stat label="Latest" value={status?.latestVersion || "—"} />
            </div>
            <Button size="sm" variant="outline" onClick={() => BrowserOpenURL(status?.downloadUrl ?? "")}>
              <Download className="size-3.5" aria-hidden /> Download latest
            </Button>
          </div>
          <p className="mt-2 text-xs text-muted-foreground">For WotLK 3.3.5a.</p>
        </Card>
      </div>

      <div>
        <h2 className="mb-2 text-sm font-semibold">Last sent</h2>
        {datasets.length === 0 ? (
          <EmptyState title="Nothing sent yet">Load rankings and choose Send to addon.</EmptyState>
        ) : (
          <Table columns={["Dataset", "Realm", "Season", "Rows", "Sent"]}>
            {datasets.map((dataset) => (
              <tr key={dataset.dataset}>
                <td className="font-medium">{dataset.label}</td>
                <td className="font-mono text-xs">{dataset.realm || "—"}</td>
                <td className="font-mono text-xs">{dataset.season || "—"}</td>
                <td className="font-mono text-xs">{dataset.rows}</td>
                <td>{formatTimeAgo(dataset.syncedAt)}</td>
              </tr>
            ))}
          </Table>
        )}
      </div>

      <AddonPathHelpModal isOpen={showHelp} onClose={() => setShowHelp(false)} />
    </div>
  );
}
