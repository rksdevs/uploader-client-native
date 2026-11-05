import { useEffect, useState } from "react";
import { toast } from "sonner";
import { CheckCircle2, FolderOpen, Loader2, LogIn } from "lucide-react";
import {
  CancelAutoUploadServerDrift,
  ReopenSignInPage,
  ResolveAutoUploadServerDrift,
} from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime";
import InstanceSelector from "../components/InstanceSelector";
import ServerDriftModal from "../components/ServerDriftModal";
import type { ServerDriftState } from "../components/ServerDriftModal";
import { Button, Modal, Select } from "../ui";
import { useApp } from "../state/AppContext";
import type { Instance } from "../types";

const ONBOARDED_KEY = "wow-logs-uploader-onboarded";

function readOnboarded(): boolean {
  try {
    return localStorage.getItem(ONBOARDED_KEY) === "1";
  } catch {
    return false;
  }
}

function writeOnboarded() {
  try {
    localStorage.setItem(ONBOARDED_KEY, "1");
  } catch {
    // Onboarding shows again next launch; harmless.
  }
}

export function SignInDialog() {
  const { signingIn, cancelSignIn } = useApp();
  if (!signingIn) return null;
  return (
    <Modal title="Finish signing in" onClose={cancelSignIn}>
      <div className="space-y-4 text-center">
        <Loader2 className="mx-auto size-8 animate-spin text-brand-text" aria-hidden />
        <p className="text-sm text-muted-foreground">Approve WoW Logs Uploader in the browser window that just opened.</p>
        <div className="flex justify-center gap-2">
          <Button variant="outline" onClick={() => void ReopenSignInPage()}>Open browser again</Button>
          <Button variant="ghost" onClick={cancelSignIn}>Cancel</Button>
        </div>
      </div>
    </Modal>
  );
}

export function InstanceDialog() {
  const { instanceFlow, queueInstances, cancelInstanceFlow, server, servers, canMarkPrivate } = useApp();
  const [processing, setProcessing] = useState(false);
  if (!instanceFlow) return null;

  function process(selected: Instance[]) {
    if (selected.length === 0) {
      toast.error("Choose at least one raid.");
      return;
    }
    setProcessing(true);
    queueInstances(selected)
      .catch((err: unknown) => toast.error("Could not queue the raids", { description: String(err) }))
      .finally(() => setProcessing(false));
  }

  return (
    <Modal title={instanceFlow.auto ? "Auto-upload: choose raids" : "Choose raids to upload"} wide onClose={cancelInstanceFlow}>
      <InstanceSelector
        instances={instanceFlow.instances}
        onProcess={process}
        onCancel={cancelInstanceFlow}
        isProcessing={processing}
        selectedServer={server}
        serverOptions={servers}
        hasMultipleDetectedServers={instanceFlow.hasMultipleDetectedServers}
        canMarkPrivate={!instanceFlow.auto && canMarkPrivate}
        defaultUnlisted={instanceFlow.instances.some((instance) => instance.unlisted)}
      />
    </Modal>
  );
}

export function DriftDialog() {
  const { servers } = useApp();
  const [drift, setDrift] = useState<ServerDriftState | null>(null);
  const [resolving, setResolving] = useState(false);

  useEffect(() => {
    const stops = [
      EventsOn("auto_upload_server_drift", (data: ServerDriftState) => {
        if (data?.defaultServer && data.detectedServers?.length) setDrift(data);
      }),
      EventsOn("auto_upload_drift_cancelled", () => {
        setDrift(null);
        setResolving(false);
      }),
    ];
    return () => stops.forEach((stop) => stop());
  }, []);

  function resolve(serverName: string) {
    setResolving(true);
    ResolveAutoUploadServerDrift(serverName)
      .then(() => setDrift(null))
      .catch((err: unknown) => toast.error("Could not continue auto-upload", { description: String(err) }))
      .finally(() => setResolving(false));
  }

  function cancel() {
    setResolving(true);
    CancelAutoUploadServerDrift()
      .then(() => setDrift(null))
      .catch((err: unknown) => toast.error("Could not cancel", { description: String(err) }))
      .finally(() => setResolving(false));
  }

  return (
    <ServerDriftModal
      isOpen={drift !== null}
      drift={drift}
      serverOptions={servers}
      isResolving={resolving}
      onUseDetected={resolve}
      onKeepDefault={() => drift && resolve(drift.defaultServer)}
      onUseSelected={resolve}
      onCancel={cancel}
    />
  );
}

export function OnboardingDialog() {
  const { logDirectory, wowDirectory, chooseLogDirectory, chooseWowDirectory, servers, server, setServer, account, signIn } = useApp();
  const [dismissed, setDismissed] = useState(readOnboarded);
  const [step, setStep] = useState(0);
  const [settled, setSettled] = useState(false);

  useEffect(() => {
    const timer = window.setTimeout(() => setSettled(true), 400);
    return () => window.clearTimeout(timer);
  }, []);

  // Upgrades from 3.x already have a folder saved and skip this.
  if (dismissed || !settled || (logDirectory && step === 0)) return null;

  function finish() {
    writeOnboarded();
    setDismissed(true);
  }

  const steps = [
    {
      title: "Welcome to WoW Logs Uploader",
      body: (
        <div className="space-y-3 text-sm text-muted-foreground">
          <p>Sign in to use your account, private logs and Premium features. You can also upload without an account.</p>
          {account?.signedIn ? (
            <p className="flex items-center gap-2 text-success"><CheckCircle2 className="size-4" aria-hidden /> Signed in</p>
          ) : (
            <Button variant="primary" className="w-full" onClick={signIn}><LogIn className="size-4" aria-hidden /> Sign in with your browser</Button>
          )}
        </div>
      ),
    },
    {
      title: "Choose your WoW folder",
      body: (
        <div className="space-y-3 text-sm text-muted-foreground">
          <p>The game folder that contains <span className="font-mono">Interface/AddOns</span>. Used to send rankings to the addon.</p>
          <p className="truncate font-mono text-xs text-foreground">{wowDirectory || "Not set"}</p>
          <Button variant="outline" onClick={() => void chooseWowDirectory()}><FolderOpen className="size-4" aria-hidden /> Choose folder</Button>
        </div>
      ),
    },
    {
      title: "Choose your combat logs folder",
      body: (
        <div className="space-y-3 text-sm text-muted-foreground">
          <p>Usually the <span className="font-mono">Logs</span> folder inside your game folder.</p>
          <p className="truncate font-mono text-xs text-foreground">{logDirectory || "Not set"}</p>
          <Button variant="outline" onClick={() => void chooseLogDirectory()}><FolderOpen className="size-4" aria-hidden /> Choose folder</Button>
        </div>
      ),
    },
    {
      title: "Choose your server",
      body: (
        <Select aria-label="Server" value={server} onChange={(event) => setServer(event.target.value)}>
          <option value="">Choose a server…</option>
          {servers.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
        </Select>
      ),
    },
  ];
  const current = steps[step];
  const last = step === steps.length - 1;

  return (
    <Modal title={current.title}>
      {current.body}
      <div className="mt-5 flex items-center justify-between">
        <span className="font-mono text-xs text-muted-foreground">Step {step + 1} of {steps.length}</span>
        <div className="flex gap-2">
          <Button variant="ghost" onClick={finish}>Skip</Button>
          <Button variant="primary" onClick={() => (last ? finish() : setStep(step + 1))}>{last ? "Done" : "Next"}</Button>
        </div>
      </div>
    </Modal>
  );
}
