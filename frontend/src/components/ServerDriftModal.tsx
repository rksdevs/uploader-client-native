import React from "react";
import { ServerOption } from "./ServerSelector";

const WARMANE_TRIO = new Set([
  "Warmane_Lordaeron",
  "Warmane_Icecrown",
  "Warmane_Onyxia",
]);

export interface ServerDriftState {
  defaultServer: string;
  detectedServers: string[];
  guidPrefixByRealm?: Record<string, string>;
  multipleDetected: boolean;
  stagingPath?: string;
  sourceLogPath?: string;
}

interface ServerDriftModalProps {
  isOpen: boolean;
  drift: ServerDriftState | null;
  serverOptions: ServerOption[];
  isResolving: boolean;
  onUseDetected: (serverName: string) => void;
  onKeepDefault: () => void;
  onUseSelected: (serverName: string) => void;
  onCancel: () => void;
}

function serverLabel(
  value: string,
  serverOptions: ServerOption[]
): string {
  if (!value) return "Unknown";
  const opt = serverOptions.find((o) => o.value === value);
  if (opt?.label) return opt.label;
  switch (value) {
    case "Warmane_Lordaeron":
      return "Warmane - Lordaeron";
    case "Warmane_Icecrown":
      return "Warmane - Icecrown";
    case "Warmane_Onyxia":
      return "Warmane - Onyxia";
    default:
      return value.replace(/_/g, " ");
  }
}

const ServerDriftModal: React.FC<ServerDriftModalProps> = ({
  isOpen,
  drift,
  serverOptions,
  isResolving,
  onUseDetected,
  onKeepDefault,
  onUseSelected,
  onCancel,
}) => {
  const [selectedRealm, setSelectedRealm] = React.useState("");

  React.useEffect(() => {
    if (drift?.detectedServers?.length) {
      setSelectedRealm(drift.detectedServers[0]);
    } else {
      setSelectedRealm("");
    }
  }, [drift]);

  if (!isOpen || !drift) return null;

  const defaultLabel = serverLabel(drift.defaultServer, serverOptions);
  const detectedLabels = drift.detectedServers.map((s) => {
    const prefix = drift.guidPrefixByRealm?.[s];
    const label = serverLabel(s, serverOptions);
    return prefix ? `${label} (GUID ${prefix})` : label;
  });

  const singleDetected =
    drift.detectedServers.length === 1 ? drift.detectedServers[0] : "";
  const showSimpleDrift =
    !drift.multipleDetected &&
    singleDetected &&
    WARMANE_TRIO.has(drift.defaultServer) &&
    WARMANE_TRIO.has(singleDetected);

  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        backgroundColor: "var(--modal-overlay)",
        backdropFilter: "blur(2px)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 1100,
      }}
    >
      <div
        style={{
          backgroundColor: "var(--modal-bg)",
          padding: "24px",
          borderRadius: "12px",
          width: "92%",
          maxWidth: "480px",
          boxShadow:
            "0 10px 25px -5px rgba(0, 0, 0, 0.1), 0 8px 10px -6px rgba(0, 0, 0, 0.1)",
          border: "1px solid var(--modal-border)",
        }}
      >
        <h3
          style={{
            marginTop: 0,
            marginBottom: "12px",
            fontSize: "1.25rem",
            fontWeight: 600,
            color: "var(--text-primary)",
          }}
        >
          Server drift detected
        </h3>
        <p
          style={{
            marginBottom: "16px",
            fontSize: "0.95rem",
            lineHeight: 1.55,
            color: "var(--text-muted)",
          }}
        >
          Your default server is <strong>{defaultLabel}</strong>, but player GUIDs
          in the <em>new combat log lines</em> about to be uploaded point to{" "}
          {drift.multipleDetected ? (
            <>multiple Warmane realms: {detectedLabels.join(", ")}.</>
          ) : (
            <>
              <strong>{detectedLabels[0] || "another realm"}</strong>.
            </>
          )}
        </p>
        <p
          style={{
            marginBottom: "20px",
            fontSize: "0.85rem",
            lineHeight: 1.5,
            color: "var(--text-muted)",
          }}
        >
          Only the staged slice (lines added since your last upload) is checked —
          not older entries from other realms in the same file.
        </p>

        {drift.multipleDetected ? (
          <div style={{ marginBottom: "20px" }}>
            <label
              htmlFor="drift-realm-select"
              style={{
                display: "block",
                marginBottom: "8px",
                fontSize: "0.9rem",
                color: "var(--text-primary)",
              }}
            >
              Which realm are these new lines from?
            </label>
            <select
              id="drift-realm-select"
              className="slice-server-select"
              value={selectedRealm}
              onChange={(e) => setSelectedRealm(e.target.value)}
              disabled={isResolving}
              style={{ width: "100%" }}
            >
              {drift.detectedServers.map((realm) => (
                <option key={realm} value={realm}>
                  {serverLabel(realm, serverOptions)}
                </option>
              ))}
            </select>
          </div>
        ) : null}

        <div
          style={{
            display: "flex",
            flexWrap: "wrap",
            gap: "10px",
            justifyContent: "flex-end",
          }}
        >
          <button
            type="button"
            onClick={onCancel}
            disabled={isResolving}
            style={{
              padding: "8px 14px",
              borderRadius: "6px",
              border: "1px solid var(--border-soft)",
              backgroundColor: "transparent",
              color: "var(--text-primary)",
              cursor: isResolving ? "not-allowed" : "pointer",
            }}
          >
            Cancel upload
          </button>
          {showSimpleDrift ? (
            <>
              <button
                type="button"
                onClick={onKeepDefault}
                disabled={isResolving}
                style={{
                  padding: "8px 14px",
                  borderRadius: "6px",
                  border: "1px solid var(--border-soft)",
                  backgroundColor: "transparent",
                  color: "var(--text-primary)",
                  cursor: isResolving ? "not-allowed" : "pointer",
                }}
              >
                Keep {defaultLabel}
              </button>
              <button
                type="button"
                onClick={() => onUseDetected(singleDetected)}
                disabled={isResolving}
                style={{
                  padding: "8px 14px",
                  borderRadius: "6px",
                  border: "none",
                  backgroundColor: "#2563eb",
                  color: "#fff",
                  cursor: isResolving ? "not-allowed" : "pointer",
                }}
              >
                Use {serverLabel(singleDetected, serverOptions)}
              </button>
            </>
          ) : (
            <button
              type="button"
              onClick={() => selectedRealm && onUseSelected(selectedRealm)}
              disabled={isResolving || !selectedRealm}
              style={{
                padding: "8px 14px",
                borderRadius: "6px",
                border: "none",
                backgroundColor: "#2563eb",
                color: "#fff",
                cursor:
                  isResolving || !selectedRealm ? "not-allowed" : "pointer",
              }}
            >
              Upload with selected realm
            </button>
          )}
        </div>
      </div>
    </div>
  );
};

export default ServerDriftModal;
