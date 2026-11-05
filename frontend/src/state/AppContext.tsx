import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import type { ReactNode } from "react";
import { toast } from "sonner";
import {
  AbandonAutoUploadInstanceSelection,
  CancelSignIn,
  CheckForUpdate,
  EnqueueAutoUploadJobs,
  EnqueueJobs,
  FetchLeaderboardSeasonsConfigJSON,
  GetAccount,
  GetAppVersion,
  GetAutoUploadWatcherStatus,
  GetAutoUploadSettings,
  GetSavedDirectory,
  GetSiteURL,
  GetTheme,
  GetUploadHistory,
  GetUploaderServers,
  GetWowDirectory,
  OpenLogPage,
  SaveDefaultServer,
  SelectDirectory,
  SelectWowDirectory,
  SetTheme,
  SignOut,
  StartMonitoringJob,
  StartMonitoringJobWithTailRefresh,
  StartSignIn,
} from "../../wailsjs/go/main/App";
import { main } from "../../wailsjs/go/models";
import { EventsOn } from "../../wailsjs/runtime";
import type { ServerOption } from "../components/ServerSelector";
import type { Instance, JobNotification } from "../types";

export type Screen = "upload" | "merge" | "auto" | "rankings" | "addon" | "settings";

export interface UploadProgress {
  stage: string;
  percent: number;
  bytesDone: number;
  bytesTotal: number;
}

export interface JobProgress {
  preprocessId: number;
  queuePosition: number | null;
  jobsCompleted: number;
  totalJobs: number;
}

export interface InstanceFlow {
  preprocessId: number;
  instances: Instance[];
  hasMultipleDetectedServers: boolean;
  /** Watch the live log after queueing, so "new since last upload" moves forward. */
  trackLiveLog: boolean;
  auto?: { stagingPath: string; sourceLogPath: string };
}

interface AppState {
  screen: Screen;
  setScreen: (screen: Screen) => void;
  theme: "light" | "dark";
  setThemeMode: (theme: "light" | "dark") => void;
  version: string;
  siteUrl: string;
  logDirectory: string;
  wowDirectory: string;
  chooseLogDirectory: () => Promise<void>;
  chooseWowDirectory: () => Promise<void>;
  servers: ServerOption[];
  server: string;
  serverId: number | null;
  setServer: (value: string) => void;
  loadServers: () => void;
  season: number | null;
  account: main.DesktopAccount | null;
  refreshAccount: () => Promise<void>;
  signingIn: boolean;
  signIn: () => void;
  cancelSignIn: () => void;
  signOut: () => void;
  update: main.UpdateInfo | null;
  history: main.HistoryEntry[];
  refreshHistory: () => void;
  watcher: main.AutoUploadWatcherStatus | null;
  refreshWatcher: () => void;
  uploadProgress: UploadProgress | null;
  setUploadProgress: (progress: UploadProgress | null) => void;
  jobProgress: JobProgress | null;
  instanceFlow: InstanceFlow | null;
  setInstanceFlow: (flow: InstanceFlow | null) => void;
  queueInstances: (selected: Instance[]) => Promise<void>;
  cancelInstanceFlow: () => void;
  monitorUpload: (preprocessId: number, trackLiveLog: boolean) => void;
  canMarkPrivate: boolean;
  isPremium: boolean;
}

const AppContext = createContext<AppState | null>(null);

export function useApp(): AppState {
  const value = useContext(AppContext);
  if (!value) throw new Error("useApp must be used inside AppProvider");
  return value;
}

function applyTheme(theme: "light" | "dark") {
  document.documentElement.classList.toggle("dark", theme === "dark");
  // legacy.css scopes its dark overrides to this class.
  document.documentElement.classList.toggle("dark-theme", theme === "dark");
}

export function AppProvider({ children }: { children: ReactNode }) {
  const [screen, setScreen] = useState<Screen>("upload");
  const [theme, setThemeState] = useState<"light" | "dark">("dark");
  const [version, setVersion] = useState("");
  const [siteUrl, setSiteUrl] = useState("");
  const [logDirectory, setLogDirectory] = useState("");
  const [wowDirectory, setWowDirectory] = useState("");
  const [servers, setServers] = useState<ServerOption[]>([]);
  const [server, setServerState] = useState("");
  const [seasonsActive, setSeasonsActive] = useState<Record<string, number>>({});
  const [account, setAccount] = useState<main.DesktopAccount | null>(null);
  const [signingIn, setSigningIn] = useState(false);
  const [update, setUpdate] = useState<main.UpdateInfo | null>(null);
  const [history, setHistory] = useState<main.HistoryEntry[]>([]);
  const [watcher, setWatcher] = useState<main.AutoUploadWatcherStatus | null>(null);
  const [uploadProgress, setUploadProgress] = useState<UploadProgress | null>(null);
  const [jobProgress, setJobProgress] = useState<JobProgress | null>(null);
  const [instanceFlow, setInstanceFlow] = useState<InstanceFlow | null>(null);
  const instanceFlowRef = useRef(instanceFlow);
  instanceFlowRef.current = instanceFlow;

  const refreshHistory = useCallback(() => {
    GetUploadHistory().then((entries) => setHistory(entries ?? [])).catch(console.error);
  }, []);

  const refreshWatcher = useCallback(() => {
    GetAutoUploadWatcherStatus().then(setWatcher).catch(console.error);
  }, []);

  const refreshAccount = useCallback(async () => {
    try {
      setAccount(await GetAccount());
    } catch (err) {
      console.error("[Account]", err);
    }
  }, []);

  const loadServers = useCallback(() => {
    GetUploaderServers()
      .then((list: ServerOption[]) => {
        if (Array.isArray(list) && list.length > 0) setServers(list);
      })
      .catch((err: unknown) => toast.error("Could not load the server list", { description: String(err) }));
  }, []);

  useEffect(() => {
    GetAppVersion().then(setVersion).catch(console.error);
    GetSiteURL().then(setSiteUrl).catch(console.error);
    GetSavedDirectory().then(setLogDirectory).catch(console.error);
    GetWowDirectory().then(setWowDirectory).catch(console.error);
    GetTheme()
      .then((saved) => {
        const next = saved === "light" ? "light" : "dark";
        setThemeState(next);
        applyTheme(next);
      })
      .catch(console.error);
    GetAutoUploadSettings()
      .then((settings) => {
        if (settings?.defaultServer) setServerState(settings.defaultServer);
      })
      .catch(console.error);
    FetchLeaderboardSeasonsConfigJSON()
      .then((json) => setSeasonsActive(JSON.parse(json)?.active ?? {}))
      .catch(() => setSeasonsActive({}));
    CheckForUpdate().then(setUpdate).catch(() => setUpdate(null));
    loadServers();
    refreshHistory();
    refreshWatcher();
    void refreshAccount();
  }, [loadServers, refreshAccount, refreshHistory, refreshWatcher]);

  useEffect(() => {
    const stops = [
      EventsOn("upload_progress", (progress: UploadProgress) => setUploadProgress(progress)),
      EventsOn("job_progress", (progress: JobProgress) => setJobProgress(progress)),
      EventsOn("job_notification", (data: JobNotification) => {
        setJobProgress(null);
        refreshHistory();
        if (data?.status === "uploaded") {
          toast.success("Log processed", {
            duration: 15000,
            action: { label: "View log", onClick: () => void OpenLogPage(data.logId) },
          });
        } else if (data?.status === "failed") {
          toast.error("Log processing failed", { description: data.error || "The server could not process this log." });
        }
      }),
      EventsOn("job_timed_out", () => {
        setJobProgress(null);
        refreshHistory();
        toast.warning("Still in the server queue", { description: "Check View all logs later." });
      }),
      EventsOn("auto_upload_status", refreshWatcher),
      EventsOn("auto_upload_skipped", (data: { message?: string }) => {
        refreshHistory();
        toast.info("Auto-upload skipped", { description: data?.message || "No raid bosses in the new log lines." });
      }),
      EventsOn("auto_upload_error", (data: { message?: string }) => {
        toast.error("Auto-upload failed", { description: data?.message || "Unknown error" });
      }),
      EventsOn("auto_upload_complete", () => {
        refreshHistory();
        toast.success("Auto-upload complete");
      }),
      EventsOn("auto_upload_premium_lost", (data: { message?: string }) => {
        void refreshAccount();
        toast.error("Auto-upload turned off", { description: data?.message || "Premium is no longer active.", duration: 15000 });
      }),
      EventsOn("auto_upload_instances_ready", (data: {
        preprocessId?: number;
        instances?: Instance[];
        hasMultipleDetectedServers?: boolean;
        defaultServer?: string;
        stagingPath?: string;
        sourceLogPath?: string;
      }) => {
        if (!data?.preprocessId || !data.instances?.length) return;
        if (data.defaultServer) setServerState(data.defaultServer);
        setInstanceFlow({
          preprocessId: data.preprocessId,
          instances: data.instances,
          hasMultipleDetectedServers: !!data.hasMultipleDetectedServers,
          trackLiveLog: false,
          auto: { stagingPath: data.stagingPath || "", sourceLogPath: data.sourceLogPath || "" },
        });
      }),
    ];
    return () => stops.forEach((stop) => stop());
  }, [refreshAccount, refreshHistory, refreshWatcher]);

  const hasProcessing = history.some((entry) => entry.status === "processing");
  useEffect(() => {
    if (!hasProcessing) return undefined;
    const timer = window.setInterval(refreshHistory, 10000);
    return () => window.clearInterval(timer);
  }, [hasProcessing, refreshHistory]);

  const setThemeMode = useCallback((next: "light" | "dark") => {
    setThemeState(next);
    applyTheme(next);
    SetTheme(next).catch(console.error);
  }, []);

  const setServer = useCallback((value: string) => {
    setServerState(value);
    SaveDefaultServer(value).catch((err: unknown) => {
      toast.error("Could not save the server", { description: String(err) });
    });
  }, []);

  const chooseLogDirectory = useCallback(async () => {
    try {
      const path = await SelectDirectory();
      if (path) setLogDirectory(path);
    } catch (err) {
      toast.error("Could not select the folder", { description: String(err) });
    }
  }, []);

  const chooseWowDirectory = useCallback(async () => {
    try {
      const path = await SelectWowDirectory();
      if (path) setWowDirectory(path);
    } catch (err) {
      toast.error("Could not select the folder", { description: String(err) });
    }
  }, []);

  const signIn = useCallback(() => {
    setSigningIn(true);
    StartSignIn()
      .then((next) => {
        setAccount(next);
        toast.success(next?.userName ? `Signed in as ${next.userName}` : "Signed in");
      })
      .catch((err: unknown) => {
        if (!String(err).includes("cancelled")) {
          toast.error("Sign-in failed", { description: String(err) });
        }
      })
      .finally(() => setSigningIn(false));
  }, []);

  const cancelSignIn = useCallback(() => {
    CancelSignIn().catch(console.error);
  }, []);

  const signOut = useCallback(() => {
    SignOut()
      .then(refreshAccount)
      .then(() => toast.success("Signed out"))
      .catch((err: unknown) => toast.error("Could not sign out", { description: String(err) }));
  }, [refreshAccount]);

  const monitorUpload = useCallback((preprocessId: number, trackLiveLog: boolean) => {
    refreshHistory();
    if (trackLiveLog) {
      StartMonitoringJobWithTailRefresh(preprocessId, logDirectory).catch(console.error);
    } else {
      StartMonitoringJob(preprocessId).catch(console.error);
    }
  }, [logDirectory, refreshHistory]);

  const queueInstances = useCallback(async (selected: Instance[]) => {
    const flow = instanceFlowRef.current;
    if (!flow) return;
    const forGo = selected.map((instance) => new main.Instance(instance));
    if (flow.auto) {
      const result = await EnqueueAutoUploadJobs(flow.preprocessId, forGo, flow.auto.sourceLogPath, flow.auto.stagingPath);
      toast.success(result);
      refreshHistory();
    } else {
      const result = await EnqueueJobs(flow.preprocessId, forGo, false);
      toast.success(result);
      monitorUpload(flow.preprocessId, flow.trackLiveLog);
    }
    setInstanceFlow(null);
  }, [monitorUpload, refreshHistory]);

  const cancelInstanceFlow = useCallback(() => {
    if (instanceFlowRef.current?.auto) {
      AbandonAutoUploadInstanceSelection().catch(console.error);
    }
    setInstanceFlow(null);
  }, []);

  const serverId = useMemo(() => servers.find((option) => option.value === server)?.id ?? null, [servers, server]);
  const season = serverId != null ? seasonsActive[String(serverId)] ?? null : null;

  const value: AppState = {
    screen, setScreen, theme, setThemeMode, version, siteUrl,
    logDirectory, wowDirectory, chooseLogDirectory, chooseWowDirectory,
    servers, server, serverId, setServer, loadServers, season,
    account, refreshAccount, signingIn, signIn, cancelSignIn, signOut,
    update, history, refreshHistory, watcher, refreshWatcher,
    uploadProgress, setUploadProgress, jobProgress,
    instanceFlow, setInstanceFlow, queueInstances, cancelInstanceFlow, monitorUpload,
    canMarkPrivate: !!account?.signedIn && account.tokenType !== "GUILD",
    isPremium: !!account?.isPremium,
  };

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}
