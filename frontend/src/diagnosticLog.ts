import { LogClientMessage } from "../wailsjs/go/main/App";

function safeLog(level: string, message: string) {
  try {
    void LogClientMessage(level, message);
  } catch {
    // Wails bindings unavailable (e.g. browser-only dev)
  }
}

export function installFrontendDiagnosticHooks() {
  window.addEventListener("error", (ev) => {
    const detail = ev.error instanceof Error ? ev.error.stack ?? ev.error.message : ev.message;
    safeLog("error", `window.error: ${detail} @ ${ev.filename}:${ev.lineno}`);
  });

  window.addEventListener("unhandledrejection", (ev) => {
    const reason = ev.reason instanceof Error ? ev.reason.stack ?? ev.reason.message : String(ev.reason);
    safeLog("error", `unhandledrejection: ${reason}`);
  });

  const origError = console.error.bind(console);
  console.error = (...args: unknown[]) => {
    origError(...args);
    safeLog("error", args.map((a) => String(a)).join(" "));
  };

  const origWarn = console.warn.bind(console);
  console.warn = (...args: unknown[]) => {
    origWarn(...args);
    safeLog("warn", args.map((a) => String(a)).join(" "));
  };
}
