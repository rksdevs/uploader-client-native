import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Eye, EyeOff, FileText, Folder, KeyRound, LogIn, LogOut, Moon, Plus, Sun, User, Users, X, ExternalLink } from "lucide-react";
import {
  CheckForUpdate,
  GetAutoUploadSettings,
  GetDesktopSettings,
  GetPremiumConfig,
  OpenDiagnosticLog,
  SavePremiumConfig,
  SetDesktopNotifications,
  SetMinimizeToTray,
  SetStartWithWindows,
} from "../../wailsjs/go/main/App";
import { main } from "../../wailsjs/go/models";
import { BrowserOpenURL } from "../../wailsjs/runtime";
import ConfirmationModal from "../components/ConfirmationModal";
import { Badge, Button, Card, Input, PageHeader, PremiumBadge, Select, Switch } from "../ui";
import { useApp } from "../state/AppContext";

function FolderRow({ label, path, onChange }: { label: string; path: string; onChange: () => void }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <div className="min-w-0">
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className="truncate font-mono text-sm" title={path}>{path || "Not set"}</p>
      </div>
      <Button size="sm" variant="outline" onClick={onChange}>Change…</Button>
    </div>
  );
}

export function SettingsScreen() {
  const {
    account, signIn, signOut, refreshAccount, siteUrl, version, theme, setThemeMode,
    logDirectory, wowDirectory, chooseLogDirectory, chooseWowDirectory,
  } = useApp();
  const [token, setToken] = useState("");
  const [tokenType, setTokenType] = useState("personal");
  const [showToken, setShowToken] = useState(false);
  const [followed, setFollowed] = useState<string[]>([]);
  const [newFollowed, setNewFollowed] = useState("");
  const [saving, setSaving] = useState(false);
  const [confirmClear, setConfirmClear] = useState(false);
  const [desktop, setDesktop] = useState<main.DesktopSettings | null>(null);
  const [deviceId, setDeviceId] = useState("");
  const [checking, setChecking] = useState(false);

  useEffect(() => {
    GetPremiumConfig()
      .then((config: Record<string, string>) => {
        setToken(config.apiToken ?? "");
        setTokenType(config.apiTokenType || "personal");
        setFollowed((config.followedPlayers ?? "").split(",").map((name) => name.trim()).filter(Boolean));
      })
      .catch(console.error);
    GetDesktopSettings().then(setDesktop).catch(console.error);
    GetAutoUploadSettings().then((settings) => setDeviceId(settings?.deviceId ?? "")).catch(console.error);
  }, []);

  function save(nextToken: string, nextType: string, nextFollowed: string[], message: string) {
    setSaving(true);
    SavePremiumConfig(nextToken, nextType, nextFollowed.join(","))
      .then(() => {
        toast.success(message);
        return refreshAccount();
      })
      .catch((err: unknown) => toast.error("Could not save", { description: String(err) }))
      .finally(() => setSaving(false));
  }

  function addFollowed() {
    const name = newFollowed.trim();
    if (!name || followed.some((existing) => existing.toLowerCase() === name.toLowerCase())) return;
    const next = [...followed, name];
    setFollowed(next);
    setNewFollowed("");
    save(token, tokenType, next, "Followed characters saved");
  }

  function removeFollowed(name: string) {
    const next = followed.filter((existing) => existing !== name);
    setFollowed(next);
    save(token, tokenType, next, "Followed characters saved");
  }

  function clearToken() {
    setConfirmClear(false);
    setToken("");
    setTokenType("personal");
    save("", "personal", followed, "API token cleared");
  }

  function behaviour(change: Partial<main.DesktopSettings>, action: Promise<void>) {
    setDesktop((current) => (current ? new main.DesktopSettings({ ...current, ...change }) : current));
    action.catch((err: unknown) => {
      toast.error("Could not save", { description: String(err) });
      GetDesktopSettings().then(setDesktop).catch(console.error);
    });
  }

  function checkUpdate() {
    setChecking(true);
    CheckForUpdate()
      .then((info) => {
        if (info.available) toast.info(`Version ${info.version} is available`, { description: "Install it from the sidebar." });
        else toast.success("You are on the latest version");
      })
      .catch((err: unknown) => toast.error("Could not check for updates", { description: String(err) }))
      .finally(() => setChecking(false));
  }

  const signedInWithSession = account?.source === "session";

  return (
    <div className="space-y-4">
      <PageHeader title="Settings" description="Account, API tokens, folders, appearance and app behaviour." />

      <Card title="Account" icon={<User className="size-4 text-brand-text" aria-hidden />}>
        {signedInWithSession ? (
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex min-w-0 items-center gap-3">
              {account?.avatarUrl && <img src={account.avatarUrl} alt="" className="size-10 rounded-full object-cover" />}
              <div className="min-w-0">
                <p className="flex items-center gap-2 font-medium">
                  <span className="truncate">{account?.userName}</span>
                  {account?.isPremium ? <PremiumBadge /> : <Badge>Free</Badge>}
                </p>
              </div>
            </div>
            <div className="flex gap-2">
              <Button variant="outline" onClick={() => BrowserOpenURL(`${siteUrl}/account/devices`)}>
                <ExternalLink className="size-4" aria-hidden /> Manage devices
              </Button>
              <Button variant="destructive" onClick={signOut}><LogOut className="size-4" aria-hidden /> Sign out</Button>
            </div>
          </div>
        ) : (
          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="text-sm text-muted-foreground">
              {account?.signedIn ? "Using an API token. Sign in to manage this device from the website." : "Sign in with Discord or Google to use your account and Premium."}
            </p>
            <Button variant="primary" onClick={signIn}><LogIn className="size-4" aria-hidden /> Sign in</Button>
          </div>
        )}
      </Card>

      <Card
        title="Advanced: API token"
        icon={<KeyRound className="size-4 text-brand-text" aria-hidden />}
        action={<span className="text-xs text-muted-foreground">Alternative to sign-in</span>}
      >
        <p className="mb-3 text-sm text-muted-foreground">
          For guild tokens or uploading without signing in. Create tokens on your website account. If you are signed in, your session is used first.
        </p>
        <div className="grid gap-2 sm:grid-cols-[180px_1fr]">
          <Select aria-label="Token type" value={tokenType} onChange={(event) => setTokenType(event.target.value)}>
            <option value="personal">Personal token</option>
            <option value="guild">Guild token</option>
          </Select>
          <div className="flex gap-2">
            <Input
              aria-label="API token"
              type={showToken ? "text" : "password"}
              className="font-mono"
              value={token}
              placeholder="Paste your token"
              onChange={(event) => setToken(event.target.value)}
            />
            <Button variant="outline" aria-label={showToken ? "Hide token" : "Show token"} onClick={() => setShowToken((value) => !value)}>
              {showToken ? <EyeOff className="size-4" aria-hidden /> : <Eye className="size-4" aria-hidden />}
            </Button>
          </div>
        </div>
        <div className="mt-3 flex justify-end gap-2">
          <Button variant="destructive" size="sm" disabled={!token} onClick={() => setConfirmClear(true)}>Clear token</Button>
          <Button variant="primary" size="sm" busy={saving} onClick={() => save(token, tokenType, followed, "API token saved")}>Save token</Button>
        </div>
      </Card>

      <Card title="Followed characters" icon={<Users className="size-4 text-brand-text" aria-hidden />} action={<PremiumBadge />}>
        <p className="mb-3 text-sm text-muted-foreground">These characters always appear in rankings sent to the addon, even outside the top rows.</p>
        <div className="flex gap-2">
          <Input
            aria-label="Character name"
            placeholder="Character name"
            value={newFollowed}
            onChange={(event) => setNewFollowed(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter") addFollowed();
            }}
          />
          <Button variant="outline" onClick={addFollowed}><Plus className="size-4" aria-hidden /> Add</Button>
        </div>
        {followed.length > 0 && (
          <div className="mt-3 flex flex-wrap gap-1.5">
            {followed.map((name) => (
              <span key={name} className="inline-flex items-center gap-1 rounded-control bg-muted px-2 py-1 text-xs font-medium">
                {name}
                <button type="button" aria-label={`Remove ${name}`} onClick={() => removeFollowed(name)}><X className="size-3" aria-hidden /></button>
              </span>
            ))}
          </div>
        )}
      </Card>

      <Card title="Folders" icon={<Folder className="size-4 text-brand-text" aria-hidden />}>
        <div className="space-y-3">
          <FolderRow label="WoW game folder (contains Interface/AddOns)" path={wowDirectory} onChange={() => void chooseWowDirectory()} />
          <FolderRow label="Combat logs folder" path={logDirectory} onChange={() => void chooseLogDirectory()} />
        </div>
      </Card>

      <div className="grid gap-4 md:grid-cols-2">
        <Card title="Theme">
          <div className="grid grid-cols-2 gap-2">
            <Button variant={theme === "dark" ? "primary" : "outline"} onClick={() => setThemeMode("dark")}><Moon className="size-4" aria-hidden /> Dark</Button>
            <Button variant={theme === "light" ? "primary" : "outline"} onClick={() => setThemeMode("light")}><Sun className="size-4" aria-hidden /> Light</Button>
          </div>
        </Card>
        <Card title="Behaviour">
          <div className="space-y-3 text-sm">
            <label className="flex items-center justify-between gap-3">
              Minimize to tray when closing
              <Switch label="Minimize to tray" checked={!!desktop?.minimizeToTray} onChange={(value) => behaviour({ minimizeToTray: value }, SetMinimizeToTray(value))} />
            </label>
            <label className="flex items-center justify-between gap-3">
              Start with Windows
              <Switch label="Start with Windows" checked={!!desktop?.startWithWindows} onChange={(value) => behaviour({ startWithWindows: value }, SetStartWithWindows(value))} />
            </label>
            <label className="flex items-center justify-between gap-3">
              Desktop notifications on upload
              <Switch label="Desktop notifications" checked={!!desktop?.desktopNotifications} onChange={(value) => behaviour({ desktopNotifications: value }, SetDesktopNotifications(value))} />
            </label>
          </div>
        </Card>
      </div>

      <Card title="About">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="min-w-0">
            <p className="font-medium">WoW Logs Uploader v{version}</p>
            <p className="truncate font-mono text-xs text-muted-foreground">Device ID: {deviceId || "—"}</p>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => void OpenDiagnosticLog()}><FileText className="size-4" aria-hidden /> Open diagnostic log</Button>
            <Button variant="outline" busy={checking} onClick={checkUpdate}>Check for updates</Button>
          </div>
        </div>
      </Card>

      <ConfirmationModal
        isOpen={confirmClear}
        title="Clear API token?"
        message="This removes the token from this computer. Your sign-in is not affected."
        confirmText="Clear token"
        cancelText="Keep token"
        onConfirm={clearToken}
        onCancel={() => setConfirmClear(false)}
        isDestructive
      />
    </div>
  );
}
