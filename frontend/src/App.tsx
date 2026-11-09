import { Toaster } from "sonner";
import { AppProvider, useApp } from "./state/AppContext";
import { Sidebar } from "./shell/Sidebar";
import { StatusStrip } from "./shell/StatusStrip";
import { DriftDialog, InstanceDialog, OnboardingDialog, SignInDialog } from "./shell/Dialogs";
import { UploadScreen } from "./screens/UploadScreen";
import { MergeScreen } from "./screens/MergeScreen";
import { SurvivorsScreen } from "./screens/SurvivorsScreen";
import { AutoUploadScreen } from "./screens/AutoUploadScreen";
import { RankingsScreen } from "./screens/RankingsScreen";
import { AddonSyncScreen } from "./screens/AddonSyncScreen";
import { SettingsScreen } from "./screens/SettingsScreen";

function CurrentScreen() {
  const { screen } = useApp();
  switch (screen) {
    case "survivors":
      return <SurvivorsScreen />;
    case "merge":
      return <MergeScreen />;
    case "auto":
      return <AutoUploadScreen />;
    case "rankings":
      return <RankingsScreen />;
    case "addon":
      return <AddonSyncScreen />;
    case "settings":
      return <SettingsScreen />;
    default:
      return <UploadScreen />;
  }
}

function Shell() {
  const { theme } = useApp();
  return (
    <div className="flex h-full min-w-0">
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <StatusStrip />
        <main className="min-h-0 flex-1 overflow-x-hidden overflow-y-auto">
          <div className="mx-auto w-full max-w-5xl px-6 py-6">
            <CurrentScreen />
          </div>
        </main>
      </div>
      <SignInDialog />
      <InstanceDialog />
      <DriftDialog />
      <OnboardingDialog />
      <Toaster position="bottom-right" theme={theme} richColors closeButton />
    </div>
  );
}

export default function App() {
  return (
    <AppProvider>
      <Shell />
    </AppProvider>
  );
}
