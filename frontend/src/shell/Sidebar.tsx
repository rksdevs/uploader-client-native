import { useState } from "react";
import { toast } from "sonner";
import { ArrowUpCircle, ChevronLeft, ChevronRight, Crown, GitMerge, LogIn, Radio, RefreshCw, Settings, Trophy, UploadCloud } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { InstallUpdate } from "../../wailsjs/go/main/App";
import { Button } from "../ui";
import { useApp } from "../state/AppContext";
import type { Screen } from "../state/AppContext";

const NAV: Array<{ screen: Screen; label: string; icon: LucideIcon }> = [
  { screen: "upload", label: "Upload", icon: UploadCloud },
  { screen: "merge", label: "Merge logs", icon: GitMerge },
  { screen: "auto", label: "Auto-upload", icon: Radio },
  { screen: "rankings", label: "Rankings", icon: Trophy },
  { screen: "addon", label: "Addon sync", icon: RefreshCw },
  { screen: "settings", label: "Settings", icon: Settings },
];

export function Sidebar() {
  const { screen, setScreen, account, signIn, update, version, season } = useApp();
  const [collapsed, setCollapsed] = useState(false);
  const [installing, setInstalling] = useState(false);

  function install() {
    setInstalling(true);
    InstallUpdate()
      .catch((err: unknown) => toast.error("Update failed", { description: String(err) }))
      .finally(() => setInstalling(false));
  }

  return (
    <nav className={`flex shrink-0 flex-col border-r border-border bg-card transition-[width] ${collapsed ? "w-14" : "w-52"}`} aria-label="Main">
      <div className="flex h-12 items-center gap-2 border-b border-border px-3">
        <div className="flex size-7 shrink-0 items-center justify-center rounded-control bg-brand text-sm font-bold text-white">W</div>
        {!collapsed && <span className="truncate text-sm font-semibold">WoW Logs Uploader</span>}
      </div>

      <div className="flex-1 space-y-1 overflow-y-auto p-2">
        {NAV.map(({ screen: target, label, icon: Icon }) => {
          const active = screen === target;
          return (
            <button
              key={target}
              type="button"
              title={label}
              aria-current={active ? "page" : undefined}
              onClick={() => setScreen(target)}
              className={`flex h-9 w-full items-center gap-3 rounded-control px-2.5 text-sm transition-colors ${
                active ? "bg-muted font-medium text-foreground" : "text-muted-foreground hover:bg-muted hover:text-foreground"
              }`}
            >
              <Icon className={`size-4 shrink-0 ${active ? "text-brand-text" : ""}`} aria-hidden />
              {!collapsed && <span className="flex-1 truncate text-left">{label}</span>}
              {!collapsed && target === "auto" && <Crown className="size-3.5 text-gold" aria-label="Premium" />}
              {!collapsed && target === "rankings" && season != null && (
                <span className="rounded bg-background px-1.5 font-mono text-[10px] text-muted-foreground">S{season}</span>
              )}
            </button>
          );
        })}
      </div>

      <div className="space-y-2 border-t border-border p-2">
        {update?.available && !collapsed && (
          <div className="flex items-center justify-between gap-2 rounded-control border border-brand-text/30 bg-accent/40 px-2 py-1.5 text-xs">
            <span className="flex min-w-0 items-center gap-1 truncate">
              <ArrowUpCircle className="size-3.5 shrink-0" aria-hidden /> v{update.version} ready
            </span>
            <button type="button" className="font-medium text-brand-text underline disabled:opacity-50" disabled={installing} onClick={install}>
              {installing ? "Installing…" : "Install"}
            </button>
          </div>
        )}

        {account?.signedIn ? (
          <button
            type="button"
            onClick={() => setScreen("settings")}
            className="flex w-full items-center gap-2 rounded-control p-1.5 text-left hover:bg-muted"
          >
            {account.avatarUrl ? (
              <img src={account.avatarUrl} alt="" className="size-7 shrink-0 rounded-full object-cover" />
            ) : (
              <span className="flex size-7 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-semibold">
                {(account.userName || "T").slice(0, 1).toUpperCase()}
              </span>
            )}
            {!collapsed && (
              <span className="min-w-0">
                <span className="block truncate text-sm font-medium">{account.userName || "API token"}</span>
                <span className={`block text-xs ${account.isPremium ? "text-gold" : "text-muted-foreground"}`}>
                  {account.isPremium ? "Premium" : "Free"}
                </span>
              </span>
            )}
          </button>
        ) : collapsed ? (
          <button type="button" title="Sign in" onClick={signIn} className="flex w-full justify-center rounded-control p-2 hover:bg-muted">
            <LogIn className="size-4" aria-hidden />
          </button>
        ) : (
          <Button variant="primary" size="sm" className="w-full" onClick={signIn}>
            <LogIn className="size-4" aria-hidden /> Sign in
          </Button>
        )}

        <div className="flex items-center justify-between px-1 text-xs text-muted-foreground">
          {!collapsed && <span className="font-mono">v{version}</span>}
          <button
            type="button"
            className="rounded p-1 hover:bg-muted"
            aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
            onClick={() => setCollapsed((value) => !value)}
          >
            {collapsed ? <ChevronRight className="size-4" aria-hidden /> : <ChevronLeft className="size-4" aria-hidden />}
          </button>
        </div>
      </div>
    </nav>
  );
}
