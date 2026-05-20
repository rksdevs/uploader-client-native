import { useState, useEffect, useMemo, useCallback } from "react";
import "./App.css";
import { Toaster } from "sonner";
import {
  BarChart3,
  Crown,
  FileText,
  FolderOpen,
  Info,
  Moon,
  Sun,
} from "lucide-react";
import {
  PreprocessLog,
  EnqueueJobs,
  SelectDirectory,
  StartMonitoringJob,
  GetSavedDirectory,
  GetUploaderServers,
  OpenAllLogsPage,
  OpenLogPage,
  GetWowDirectory,
  SelectWowDirectory,
  GetPremiumConfig,
  SavePremiumConfig,
  GetTheme,
  SetTheme,
  GetAppVersion,
} from "../wailsjs/go/main/App";
import { main } from "../wailsjs/go/models";
import { EventsOn, BrowserOpenURL } from "../wailsjs/runtime";
import { toast } from "sonner";

import DirectorySelector from "./components/DirectorySelector";
import ServerSelector, { ServerOption } from "./components/ServerSelector";
import UploadButton from "./components/UploadButton";
import InstanceSelector from "./components/InstanceSelector";
import ConfirmationModal from "./components/ConfirmationModal";
import AddonPathHelpModal from "./components/AddonPathHelpModal";
import RankingsBrowser from "./components/RankingsBrowser";
import RosterRankingsBrowser from "./components/RosterRankingsBrowser";
import GuildRankingsBrowser from "./components/GuildRankingsBrowser";
import { Instance, JobNotification } from "./types";

function App() {
  const [logDirectory, setLogDirectory] = useState<string>("");
  const [selectedServer, setSelectedServer] = useState<string>("");
  const [statusMessage, setStatusMessage] = useState<string>(
    "Loading saved settings..."
  );
  const [isProcessing, setIsProcessing] = useState<boolean>(false);
  const [view, setView] = useState<"upload" | "select">("upload");
  const [preprocessId, setPreprocessId] = useState<number | null>(null);
  const [instances, setInstances] = useState<Instance[]>([]);
  const [hasMultipleDetectedServers, setHasMultipleDetectedServers] =
    useState<boolean>(false);
  const [serverOptions, setServerOptions] = useState<ServerOption[]>([]);
  const [wowDirectory, setWowDirectory] = useState<string>("");
  const [showPremiumSettings, setShowPremiumSettings] = useState<boolean>(false);
  const [apiToken, setApiToken] = useState<string>("");
  const [apiTokenType, setApiTokenType] = useState<string>("personal");
  const [followedPlayers, setFollowedPlayers] = useState<string>("");
  const [isSavingPremium, setIsSavingPremium] = useState<boolean>(false);
  const [showClearConfirm, setShowClearConfirm] = useState<boolean>(false);
  const [showAddonHelp, setShowAddonHelp] = useState<boolean>(false);
  const [theme, setTheme] = useState<string>("light");
  const [appVersion, setAppVersion] = useState<string>("3.1.0");

  const loadServers = useCallback((silent?: boolean) => {
    GetUploaderServers()
      .then((servers: ServerOption[]) => {
        if (!Array.isArray(servers) || servers.length === 0) {
          if (!silent) {
            toast.warning("Server list is empty.");
          }
          return;
        }
        setServerOptions(servers);
        setSelectedServer((prev) =>
          prev && !servers.some((s) => s.value === prev) ? "" : prev
        );
        if (!silent) {
          toast.success(`Server list updated (${servers.length}).`);
        }
      })
      .catch((err: unknown) => {
        console.error("[React App] Error fetching uploader servers:", err);
        toast.error("Failed to fetch server list.", {
          description: String(err),
        });
      });
  }, []);

  useEffect(() => {
    GetAppVersion()
      .then((v: string) => {
        if (v?.trim()) setAppVersion(v.trim());
      })
      .catch(() => {});
    GetSavedDirectory()
      .then((savedPath: string) => {
        if (savedPath) {
          setLogDirectory(savedPath);
          setStatusMessage(`Monitoring logs in: ${savedPath}`);
        } else {
          setStatusMessage("Please select your WoW Logs directory to begin.");
        }
      })
      .catch((err: unknown) => {
        console.error("[React App] Error getting saved directory:", err);
        setStatusMessage(
          "Could not load settings. Please select your WoW Logs directory."
        );
      });

    loadServers(true);

    GetWowDirectory()
      .then((savedPath: string) => {
        if (savedPath) {
          setWowDirectory(savedPath);
        }
      })
      .catch((err: unknown) => {
        console.error("[React App] Error loading WoW directory path:", err);
      });

    GetPremiumConfig()
      .then((cfg: Record<string, string>) => {
        if (cfg.apiToken) setApiToken(cfg.apiToken);
        if (cfg.apiTokenType) setApiTokenType(cfg.apiTokenType);
        if (cfg.followedPlayers) setFollowedPlayers(cfg.followedPlayers);
      })
      .catch((err: unknown) => {
        console.error("[React App] Error loading premium config:", err);
      });

    GetTheme()
      .then((savedTheme: string) => {
        if (savedTheme) setTheme(savedTheme);
      })
      .catch((err: unknown) => {
        console.error("[React App] Error loading theme:", err);
      });
  }, [loadServers]);

  useEffect(() => {
    const cleanup = EventsOn("job_notification", (data: JobNotification) => {
      if (data && data.status === "uploaded") {
        toast.success("Log processed successfully!", {
          duration: Infinity,
          action: {
            label: "View Log",
            onClick: () => {
              OpenLogPage(data.logId);
            },
          },
        });
      } else if (data && data.status === "failed") {
        toast.error("Log processing failed", {
          description: data.error || "An unknown error occurred.",
          duration: 15000,
        });
      }
    });

    return () => {
      cleanup();
    };
  }, []);

  const handleSelectDirectory = () => {
    SelectDirectory()
      .then((selectedPath) => {
        if (selectedPath) {
          setLogDirectory(selectedPath);
          setStatusMessage(`Monitoring logs in: ${selectedPath}`);
        }
      })
      .catch((err) => {
        setStatusMessage("Error: Could not select directory.");
        toast.error("Could not select directory", { description: String(err) });
      });
  };

  const handleSelectWowDirectory = () => {
    SelectWowDirectory()
      .then((selectedPath: string) => {
        if (selectedPath) {
          setWowDirectory(selectedPath);
          toast.success("WoW Directory selected.");
        }
      })
      .catch((err: unknown) => {
        toast.error("Could not select WoW Directory", {
          description: String(err),
        });
      });
  };

  const handleSavePremiumConfig = () => {
    setIsSavingPremium(true);
    SavePremiumConfig(apiToken, apiTokenType, followedPlayers)
      .then(() => {
        toast.success("Premium settings saved!", {
          description: apiToken
            ? `Token type: ${apiTokenType}. Followed players: ${followedPlayers || "none"}`
            : "No token set — running in free mode.",
        });
        setShowPremiumSettings(false);
      })
      .catch((err) => {
        toast.error("Failed to save premium settings", { description: String(err) });
      })
      .finally(() => setIsSavingPremium(false));
  };

  const handleClearPremiumConfig = () => {
    setShowClearConfirm(true);
  };

  const confirmClear = () => {
    setShowClearConfirm(false);
    setIsSavingPremium(true);
    SavePremiumConfig("", "personal", "")
      .then(() => {
        setApiToken("");
        setApiTokenType("personal");
        setFollowedPlayers("");
        toast.success("Premium settings cleared.");
        setShowPremiumSettings(false);
      })
      .catch((err) => {
        toast.error("Failed to clear premium settings", { description: String(err) });
      })
      .finally(() => setIsSavingPremium(false));
  };

  const toggleTheme = () => {
    const newTheme = theme === "light" ? "dark" : "light";
    setTheme(newTheme);
    SetTheme(newTheme).catch((err: unknown) => {
      console.error("[React App] Failed to save theme:", err);
    });
  };

  const handlePreprocess = () => {
    if (!logDirectory || !selectedServer) {
      toast.error("Please select a log directory and a server first.");
      return;
    }
    setIsProcessing(true);
    setStatusMessage("Scanning log for raid instances...");

    PreprocessLog(logDirectory, selectedServer)
      .then((response) => {
        if (response.autoQueued) {
          toast.success(response.message);
          StartMonitoringJob(response.preprocessId);
          resetToUploadView();
        } else {
          setInstances(response.instances);
          setPreprocessId(response.preprocessId);
          setHasMultipleDetectedServers(!!response.hasMultipleDetectedServers);
          setView("select");
        }
      })
      .catch((err) => {
        setStatusMessage(`Error: ${err}`);
        toast.error("Failed to preprocess log", { description: String(err) });
      })
      .finally(() => {
        setIsProcessing(false);
      });
  };

  const handleEnqueue = (selectedInstances: Instance[]) => {
    if (!preprocessId || selectedInstances.length === 0) {
      toast.error("No instances were selected.");
      return;
    }
    setIsProcessing(true);
    setStatusMessage("Queuing selected instances for processing...");

    const instancesForGo = selectedInstances.map(
      (inst) => new main.Instance(inst)
    );

    EnqueueJobs(preprocessId, instancesForGo)
      .then((result) => {
        toast.success(result);
        StartMonitoringJob(preprocessId);
        resetToUploadView();
      })
      .catch((err) => {
        setStatusMessage(`Error: ${err}`);
        toast.error("Failed to queue jobs", { description: String(err) });
      })
      .finally(() => {
        setIsProcessing(false);
      });
  };

  const handleViewAllLogs = () => {
    try {
      OpenAllLogsPage();
    } catch (err) {
      toast.error("Failed to open all logs page", { description: String(err) });
    }
  };

  const selectedServerNumericId = useMemo(() => {
    const opt = serverOptions.find((o) => o.value === selectedServer);
    return opt?.id ?? null;
  }, [serverOptions, selectedServer]);

  const activityBannerMessage = useMemo(() => {
    const trimmedLogs = logDirectory.trim();
    const redundantMonitoring =
      trimmedLogs !== "" &&
      statusMessage === `Monitoring logs in: ${logDirectory}`;
    if (!statusMessage || redundantMonitoring) {
      return null;
    }
    return statusMessage;
  }, [statusMessage, logDirectory]);

  const resetToUploadView = () => {
    setView("upload");
    setInstances([]);
    setPreprocessId(null);
    setHasMultipleDetectedServers(false);
    if (logDirectory) {
      setStatusMessage(`Monitoring logs in: ${logDirectory}`);
    } else {
      setStatusMessage("Please select your WoW Logs directory to begin.");
    }
    loadServers(true);
  };

  const themeClass = theme === "dark" ? "dark-theme" : "light-theme";

  return (
    <div id="App" className={`app-root ${themeClass}`}>
      <Toaster
        richColors
        closeButton
        position="top-center"
        theme={theme === "dark" ? "dark" : "light"}
      />
      <div className="app-gradient-shell">
        <div className="app-inner">
          <header className="app-header">
            <div className="app-header-brand">
              <div className="app-logo-tile" aria-hidden>
                <BarChart3 size={28} strokeWidth={2} />
              </div>
              <div>
                <h1 className="app-title">WoW Logs Uploader - V {appVersion}</h1>
                <p className="app-subtitle">Combat log analysis made easy</p>
              </div>
            </div>
            <button
              type="button"
              onClick={toggleTheme}
              className="theme-icon-btn"
              title={`Switch to ${theme === "light" ? "dark" : "light"} mode`}
              aria-label={`Switch to ${theme === "light" ? "dark" : "light"} mode`}
            >
              {theme === "light" ? (
                <Moon size={20} strokeWidth={2} className="theme-icon-moon" />
              ) : (
                <Sun size={20} strokeWidth={2} className="theme-icon-sun" />
              )}
            </button>
          </header>

          {view === "upload" ? (
            <>
              <div className="redesign-card monitored-paths-card">
                <div className="paths-block">
                  <div className="paths-block__head">
                    <span className="paths-block__label">Logs monitored</span>
                  </div>
                  <div className="mono-path-box mono-path-box--tight">
                    {logDirectory.trim() !== "" ? (
                      logDirectory
                    ) : (
                      <span className="mono-path-placeholder">
                        No combat logs folder selected yet. Use step 1 below.
                      </span>
                    )}
                  </div>
                </div>

                <div className="paths-block paths-block--wow">
                  <div className="paths-block__head">
                    <Info size={16} strokeWidth={2} className="card-head-icon" aria-hidden />
                    <span className="paths-block__label">WoW directory monitored</span>
                    <button
                      type="button"
                      className="help-icon-btn"
                      onClick={() => setShowAddonHelp(true)}
                      title="Where to find your WoW install"
                      aria-label="Help: WoW folder path"
                    >
                      <svg
                        width="16"
                        height="16"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        strokeWidth="2.5"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                      >
                        <circle cx="12" cy="12" r="10" />
                        <line x1="12" y1="8" x2="12" y2="8.01" />
                        <line x1="12" y1="12" x2="12" y2="16" />
                      </svg>
                    </button>
                    {apiToken ? (
                      <span className="premium-chip-inline">
                        <Crown size={14} strokeWidth={2} aria-hidden /> Premium
                      </span>
                    ) : null}
                  </div>
                  <p className="paths-block__hint">
                    Game install root. Rankings and addon files sync under Interface/AddOns.
                  </p>
                  <div className="mono-path-box mono-path-box--tight">
                    {wowDirectory ? (
                      wowDirectory
                    ) : (
                      <span className="mono-path-placeholder">
                        &lt;wow-directory&gt; (e.g. E:\World of Warcraft 3.3.5a)
                      </span>
                    )}
                  </div>
                  <button
                    type="button"
                    className="btn-surface btn-with-icon paths-block__action"
                    onClick={handleSelectWowDirectory}
                    disabled={isProcessing}
                  >
                    <FolderOpen size={18} strokeWidth={2} aria-hidden />
                    {!wowDirectory ? "Link WoW directory" : "Change WoW directory"}
                  </button>
                </div>

                {activityBannerMessage ? (
                  <div className="paths-activity-strip" role="status">
                    {activityBannerMessage}
                  </div>
                ) : null}
              </div>

              <div className="redesign-card setup-card">
                <div className="setup-step">
                  <div className="step-badge">1</div>
                  <div className="setup-step-body">
                    <span className="step-title">Combat logs folder</span>
                    <DirectorySelector
                      onSelect={handleSelectDirectory}
                      disabled={isProcessing}
                    />
                  </div>
                </div>
                <div className="setup-step">
                  <div className="step-badge">2</div>
                  <div className="setup-step-body setup-step-body--grow">
                    <span className="step-title">Choose your server</span>
                    <div className="server-row">
                      <ServerSelector
                        selectedValue={selectedServer}
                        onSelect={setSelectedServer}
                        disabled={isProcessing}
                        serverOptions={serverOptions}
                      />
                      <button
                        type="button"
                        className="btn-outline-refresh"
                        onClick={() => loadServers(false)}
                        disabled={isProcessing}
                        title="Re-fetch servers from the API (use after adding realms in the database)"
                      >
                        Refresh
                      </button>
                    </div>
                  </div>
                </div>
              </div>

              <button
                type="button"
                className="premium-strip-toggle"
                onClick={() => setShowPremiumSettings(!showPremiumSettings)}
              >
                <Crown size={16} strokeWidth={2} aria-hidden />
                <span className="premium-strip-toggle__text">
                  {showPremiumSettings ? "Hide premium settings" : "Premium settings"}
                </span>
                {apiToken ? <span className="premium-strip-active">Active</span> : null}
              </button>

              {showPremiumSettings ? (
                <div className="redesign-card premium-panel-card">
                  <p className="premium-info">
                    Enter your API token from <strong>wow-logs.co.in</strong> to enable premium
                    features (performance trends, custom player list).
                  </p>

                  <div className="token-warning">
                    <strong>Do not share your generated token.</strong> Treat it as a secret bound
                    to your account. Sharing it publicly may compromise your account.
                  </div>

                  <div className="form-group">
                    <label>Token type</label>
                    <select
                      value={apiTokenType}
                      onChange={(e) => setApiTokenType(e.target.value)}
                      className="form-select"
                    >
                      <option value="personal">Personal</option>
                      <option value="guild">Guild</option>
                    </select>
                  </div>

                  <div className="form-group">
                    <label>API token</label>
                    <input
                      type="password"
                      className="form-input"
                      value={apiToken}
                      onChange={(e) => setApiToken(e.target.value)}
                      placeholder="Paste your token from your profile / guild dashboard…"
                    />
                  </div>

                  <div className="form-group">
                    <label>
                      Followed players{" "}
                      <span className="label-hint">(comma-separated, max 5)</span>
                    </label>
                    <input
                      type="text"
                      className="form-input"
                      value={followedPlayers}
                      onChange={(e) => setFollowedPlayers(e.target.value)}
                      placeholder="e.g. Arthas, Sylvanas, Thrall"
                    />
                  </div>

                  <div className="premium-actions">
                    <button
                      type="button"
                      className="btn-gradient-primary"
                      onClick={handleSavePremiumConfig}
                      disabled={isSavingPremium}
                    >
                      {isSavingPremium ? "Saving…" : "Save settings"}
                    </button>
                    {apiToken ? (
                      <button
                        type="button"
                        className="btn-danger-ghost"
                        onClick={handleClearPremiumConfig}
                        disabled={isSavingPremium}
                      >
                        Clear data
                      </button>
                    ) : null}
                  </div>
                </div>
              ) : null}

              <div className="action-grid-2">
                <UploadButton
                  onUpload={handlePreprocess}
                  disabled={isProcessing || !logDirectory || !selectedServer}
                  isProcessing={isProcessing}
                />
                <button
                  type="button"
                  className="btn-surface btn-with-icon"
                  onClick={handleViewAllLogs}
                  disabled={isProcessing}
                >
                  <FileText size={18} strokeWidth={2} aria-hidden />
                  View all logs
                </button>
              </div>

              <RankingsBrowser
                selectedServer={selectedServer}
                serverNumericId={selectedServerNumericId}
                wowDirectory={wowDirectory}
                disabled={isProcessing}
                theme={theme === "dark" ? "dark" : "light"}
              />

              <RosterRankingsBrowser
                serverNumericId={selectedServerNumericId}
                wowDirectory={wowDirectory}
                disabled={isProcessing}
                theme={theme === "dark" ? "dark" : "light"}
              />

              <GuildRankingsBrowser
                wowDirectory={wowDirectory}
                disabled={isProcessing}
                theme={theme === "dark" ? "dark" : "light"}
              />
            </>
          ) : (
            <div className="redesign-card instance-flow-card">
              <InstanceSelector
                instances={instances}
                onProcess={handleEnqueue}
                onCancel={resetToUploadView}
                isProcessing={isProcessing}
                selectedServer={selectedServer}
                serverOptions={serverOptions}
                hasMultipleDetectedServers={hasMultipleDetectedServers}
              />
            </div>
          )}

          <footer className="app-footer-links">
            <a
              href="#"
              className="app-footer-links__a"
              onClick={(e) => {
                e.preventDefault();
                BrowserOpenURL("https://github.com/rksdevs/WowLogsAddon/releases/latest");
              }}
            >
              Download latest WoW addon
            </a>
          </footer>
        </div>
      </div>

      <ConfirmationModal
        isOpen={showClearConfirm}
        title="Clear Premium Settings?"
        message="Are you sure you want to clear all premium settings? This will remove your API token and followed players list from this device."
        confirmText="Clear Everything"
        cancelText="Keep Settings"
        onConfirm={confirmClear}
        onCancel={() => setShowClearConfirm(false)}
        isDestructive={true}
      />

      <AddonPathHelpModal
        isOpen={showAddonHelp}
        onClose={() => setShowAddonHelp(false)}
      />
    </div>
  );
}

export default App;
