import { useState, useEffect } from "react";
import "./App.css";
import {
  PreprocessLog,
  EnqueueJobs,
  SelectDirectory,
  StartMonitoringJob,
  GetSavedDirectory,
} from "../wailsjs/go/main/App";
import { main } from "../wailsjs/go/models";
import { EventsOn } from "../wailsjs/runtime";
import { toast } from "sonner";

import StatusDisplay from "./components/StatusDisplay";
import DirectorySelector from "./components/DirectorySelector";
import ServerSelector from "./components/ServerSelector";
import UploadButton from "./components/UploadButton";
import InstanceSelector from "./components/InstanceSelector";
import { PreprocessResponse, Instance, JobNotification } from "./types";

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

  useEffect(() => {
    GetSavedDirectory()
      .then((savedPath: any) => {
        if (savedPath) {
          setLogDirectory(savedPath);
          setStatusMessage(`Monitoring logs in: ${savedPath}`);
        } else {
          setStatusMessage("Please select your WoW Logs directory to begin.");
        }
      })
      .catch((err: any) => {
        console.error("[React App] Error getting saved directory:", err);
        setStatusMessage(
          "Could not load settings. Please select your WoW Logs directory."
        );
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
              const reportUrl = `${data.viewLogURL}/${data.logId}`;
              window.open(reportUrl, "_blank");
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
        toast.error("Could not select directory", { description: err });
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
          setView("select");
        }
      })
      .catch((err) => {
        setStatusMessage(`Error: ${err}`);
        toast.error("Failed to preprocess log", { description: err });
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
        toast.error("Failed to queue jobs", { description: err });
      })
      .finally(() => {
        setIsProcessing(false);
      });
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
              disabled={isProcessing}
            />
            <ServerSelector
              selectedValue={selectedServer}
              onSelect={setSelectedServer}
              disabled={isProcessing}
            />
            <UploadButton
              onUpload={handlePreprocess}
              disabled={isProcessing || !logDirectory || !selectedServer}
              isProcessing={isProcessing}
            />
          </>
        ) : (
          <InstanceSelector
            instances={instances}
            onProcess={handleEnqueue}
            onCancel={resetToUploadView}
            isProcessing={isProcessing}
          />
        )}
      </div>
    </div>
  );
}

export default App;
