import { useState } from "react";
import { Shield, Trophy, Users } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import RankingsBrowser from "../components/RankingsBrowser";
import RosterRankingsBrowser from "../components/RosterRankingsBrowser";
import GuildRankingsBrowser from "../components/GuildRankingsBrowser";
import { Badge, Callout, PageHeader } from "../ui";
import { useApp } from "../state/AppContext";

type Tab = "leaderboard" | "roster" | "guild";

const TABS: Array<{ id: Tab; label: string; icon: LucideIcon }> = [
  { id: "leaderboard", label: "Leaderboard", icon: Trophy },
  { id: "roster", label: "Raid / Party", icon: Users },
  { id: "guild", label: "Guild", icon: Shield },
];

export function RankingsScreen() {
  const { server, serverId, wowDirectory, theme, season } = useApp();
  const [tab, setTab] = useState<Tab>("leaderboard");

  return (
    <div>
      <PageHeader
        title="Rankings"
        badge={season != null ? <Badge tone="brand">Season {season}</Badge> : undefined}
        description="Browse the same rankings as the website and send them to the in-game addon."
      />

      {!server && <div className="mb-4"><Callout>Choose a server at the top to load rankings.</Callout></div>}

      <div className="mb-4 flex gap-1 border-b border-border" role="tablist">
        {TABS.map(({ id, label, icon: Icon }) => (
          <button
            key={id}
            type="button"
            role="tab"
            aria-selected={tab === id}
            onClick={() => setTab(id)}
            className={`-mb-px flex items-center gap-2 border-b-2 px-3 py-2 text-sm ${
              tab === id ? "border-brand-text font-medium text-foreground" : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            <Icon className="size-4" aria-hidden /> {label}
          </button>
        ))}
      </div>

      {tab === "leaderboard" && (
        <RankingsBrowser selectedServer={server} serverNumericId={serverId} wowDirectory={wowDirectory} disabled={false} theme={theme} />
      )}
      {tab === "roster" && (
        <RosterRankingsBrowser serverNumericId={serverId} wowDirectory={wowDirectory} disabled={false} theme={theme} />
      )}
      {tab === "guild" && <GuildRankingsBrowser wowDirectory={wowDirectory} disabled={false} theme={theme} />}
    </div>
  );
}
