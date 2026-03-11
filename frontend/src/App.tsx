import { useState, useEffect } from "react";
import "./App.css";
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
  UpdateAddonRankings,
  GetPremiumConfig,
  SavePremiumConfig,
} from "../wailsjs/go/main/App";
import { main } from "../wailsjs/go/models";
import {
  EventsOn,
  EventsOff,
  LogInfo,
  BrowserOpenURL,
} from "../wailsjs/runtime";
import { toast } from "sonner";

import StatusDisplay from "./components/StatusDisplay";
import DirectorySelector from "./components/DirectorySelector";
import ServerSelector, { ServerOption } from "./components/ServerSelector";
import UploadButton from "./components/UploadButton";
import InstanceSelector from "./components/InstanceSelector";
import ConfirmationModal from "./components/ConfirmationModal";
import AddonPathHelpModal from "./components/AddonPathHelpModal";
import { Instance, JobNotification } from "./types";

function App() {
  const [logDirectory, setLogDirectory] = useState<string>("");
  const [selectedServer, setSelectedServer] = useState<string>("");
  const [statusMessage, setStatusMessage] = useState<string>(
    "Loading saved settings..."
  );
  const [isProcessing, setIsProcessing] = useState<boolean>(false);
  const [isUpdatingRankings, setIsUpdatingRankings] = useState<boolean>(false);
  const [view, setView] = useState<"upload" | "select">("upload");
  const [preprocessId, setPreprocessId] = useState<number | null>(null);
  const [instances, setInstances] = useState<Instance[]>([]);
  const [serverOptions, setServerOptions] = useState<ServerOption[]>([]);
  const [wowDirectory, setWowDirectory] = useState<string>("");
  const [showPremiumSettings, setShowPremiumSettings] = useState<boolean>(false);
  const [apiToken, setApiToken] = useState<string>("");
  const [apiTokenType, setApiTokenType] = useState<string>("personal");
  const [followedPlayers, setFollowedPlayers] = useState<string>("");
  const [isSavingPremium, setIsSavingPremium] = useState<boolean>(false);
  const [showClearConfirm, setShowClearConfirm] = useState<boolean>(false);
  const [showAddonHelp, setShowAddonHelp] = useState<boolean>(false);

  useEffect(() => {
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

    GetUploaderServers()
      .then((servers: ServerOption[]) => {
        if (!Array.isArray(servers) || servers.length === 0) {
          return;
        }
        setServerOptions(servers);
      })
      .catch((err: unknown) => {
        console.error("[React App] Error fetching uploader servers:", err);
        toast.error("Failed to fetch latest server list.");
      });

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
  }, []);

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

  const handleUpdateRankings = () => {
    if (!selectedServer) {
      toast.error("Select a server before updating rankings.");
      return;
    }

    setIsUpdatingRankings(true);
    UpdateAddonRankings(selectedServer, 0)
      .then((message) => {
        toast.success(message, {
          description: "Run /reload in-game to load fresh rankings.",
          duration: 12000,
        });
      })
      .catch((err) => {
        toast.error("Failed to update addon rankings", {
          description: String(err),
        });
      })
      .finally(() => setIsUpdatingRankings(false));
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

  const resetToUploadView = () => {
    setView("upload");
    setInstances([]);
    setPreprocessId(null);
    if (logDirectory) {
      setStatusMessage(`Monitoring logs in: ${logDirectory}`);
    } else {
      setStatusMessage("Please select your WoW Logs directory to begin.");
    }
  };

  return (
    <div id="App">
      <div className="container">
        <h1>WoW Logs Uploader</h1>
        {view === "upload" ? (
          <>
            <StatusDisplay message={statusMessage} />
            <DirectorySelector
              onSelect={handleSelectDirectory}
              disabled={isProcessing || isUpdatingRankings}
            />
            <ServerSelector
              selectedValue={selectedServer}
              onSelect={setSelectedServer}
              disabled={isProcessing || isUpdatingRankings}
              serverOptions={serverOptions}
            />

            <div className="addon-path-hint" style={{ display: "flex", alignItems: "center", gap: "6px" }}>
              WoW Folder (Auto-Syncs all accounts): {wowDirectory || <span style={{ opacity: 0.7, fontSize: '11px' }}>&lt;wow-directory&gt; (e.g. E:\World of Warcraft 3.3.5a)</span>}
              <button 
                onClick={() => setShowAddonHelp(true)}
                style={{ 
                  background: "none", 
                  border: "none", 
                  cursor: "pointer", 
                  padding: "0",
                  lineHeight: 0,
                  opacity: 0.6,
                  transition: "opacity 0.15s",
                }}
                title="How to find this file"
                onMouseEnter={e => (e.currentTarget.style.opacity = "1")}
                onMouseLeave={e => (e.currentTarget.style.opacity = "0.6")}
              >
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="#6366f1" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                  <circle cx="12" cy="12" r="10"/>
                  <line x1="12" y1="8" x2="12" y2="8.01"/>
                  <line x1="12" y1="12" x2="12" y2="16"/>
                </svg>
              </button>
              {apiToken && <span className="premium-badge"> ★ Premium</span>}
            </div>

            <div className="premium-settings-section">
              <button
                className="btn btn-ghost btn-sm"
                onClick={() => setShowPremiumSettings(!showPremiumSettings)}
              >
                {showPremiumSettings ? "▲ Hide Premium Settings" : "▼ Premium Settings"}
              </button>

              {showPremiumSettings && (
                <div className="premium-panel">
                  <p className="premium-info">
                    Enter your API token from <strong>wow-logs.co.in</strong> to enable premium features
                    (performance trends, custom player list).
                  </p>
                  
                  <div className="token-warning" style={{ 
                    fontSize: "0.8rem", 
                    color: "#854d0e", 
                    marginBottom: "1rem", 
                    backgroundColor: "#fefce8", 
                    padding: "10px", 
                    borderRadius: "6px",
                    border: "1px solid #fef08a"
                  }}>
                    ⚠️ <strong>Do not share your generated token.</strong> Treat it as a Secret key bound to your account. Sharing it publicly might lead to compromising your account details.
                  </div>

                  <div className="form-group">
                    <label>Token Type</label>
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
                    <label>API Token</label>
                    <input
                      type="password"
                      className="form-input"
                      value={apiToken}
                      onChange={(e) => setApiToken(e.target.value)}
                      placeholder="Paste your token from your profile / guild dashboard..."
                    />
                  </div>

                  <div className="form-group">
                    <label>Followed Players <span className="label-hint">(comma-separated, max 5)</span></label>
                    <input
                      type="text"
                      className="form-input"
                      value={followedPlayers}
                      onChange={(e) => setFollowedPlayers(e.target.value)}
                      placeholder="e.g. Arthas, Sylvanas, Thrall"
                    />
                  </div>

                  <div style={{ display: "flex", gap: "10px" }}>
                    <button
                      className="btn btn-primary"
                      onClick={handleSavePremiumConfig}
                      disabled={isSavingPremium}
                      style={{ flex: 1 }}
                    >
                      {isSavingPremium ? "Saving..." : "Save Settings"}
                    </button>
                    {apiToken && (
                      <button
                        className="btn btn-secondary"
                        onClick={handleClearPremiumConfig}
                        disabled={isSavingPremium}
                        style={{ flex: 1, backgroundColor: "#fee2e2", color: "#991b1b", border: "1px solid #fecaca" }}
                      >
                        Clear Data
                      </button>
                    )}
                  </div>
                </div>
              )}
            </div>

            <div className="action-row">
              <button
                className="btn btn-secondary w-full select-wow-btn"
                onClick={handleSelectWowDirectory}
                disabled={isProcessing || isUpdatingRankings}
              >
                {!wowDirectory
                    ? "Link WoW Directory"
                    : "Change WoW Directory"}
              </button>
              <button
                className="upload-button"
                onClick={handleUpdateRankings}
                disabled={
                  isProcessing ||
                  isUpdatingRankings ||
                  !selectedServer ||
                  !wowDirectory
                }
              >
                {isUpdatingRankings ? "Updating Rankings..." : "Update Rankings"}
              </button>
            </div>

            <div className="action-row">
              <UploadButton
                onUpload={handlePreprocess}
                disabled={isProcessing || !logDirectory || !selectedServer}
                isProcessing={isProcessing}
              />
              <button
                className="btn btn-secondary"
                onClick={handleViewAllLogs}
                disabled={isProcessing || isUpdatingRankings}
              >
                View All Logs
              </button>
            </div>
          </>
        ) : (
          <InstanceSelector
            instances={instances}
            onProcess={handleEnqueue}
            onCancel={resetToUploadView}
            isProcessing={isProcessing}
          />
        )}

        <div className="footer-links" style={{ 
          marginTop: "30px", 
          textAlign: "center", 
          borderTop: "1px solid #eef2f6", 
          paddingTop: "15px",
          display: "flex",
          justifyContent: "center",
          gap: "20px",
          opacity: 0.9
        }}>
          <a
            href="#"
            style={{ 
              color: "#312e81", 
              fontSize: "12px", 
              fontWeight: "600",
              textDecoration: "underline",
              letterSpacing: "0.3px"
            }}
            onClick={(e) => {
              e.preventDefault();
              BrowserOpenURL("https://github.com/rksdevs/wow-logs-addon/releases/latest");
            }}
          >
            Download Latest WoW Addon
          </a>
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
