import React, { useState } from "react";
import { Instance } from "../types";
import { ServerOption } from "./ServerSelector";

interface InstanceSelectorProps {
  instances: Instance[];
  onProcess: (selectedInstances: Instance[]) => void;
  onCancel: () => void;
  isProcessing: boolean;
  selectedServer: string;
  serverOptions: ServerOption[];
  hasMultipleDetectedServers: boolean;
}

function InstanceSelector({
  instances,
  onProcess,
  onCancel,
  isProcessing,
  selectedServer,
  serverOptions,
  hasMultipleDetectedServers,
}: InstanceSelectorProps) {
  const [selectedIndices, setSelectedIndices] = useState<Set<number>>(
    new Set()
  );
  const [instanceServerSelection, setInstanceServerSelection] = useState<
    Record<number, string>
  >(() => {
    const initial: Record<number, string> = {};
    instances.forEach((instance, index) => {
      initial[index] =
        instance.serverName ||
        instance.preview?.detectedServerName ||
        selectedServer;
    });
    return initial;
  });
  const [instanceServerVerified, setInstanceServerVerified] = useState<
    Record<number, boolean>
  >(() => {
    const initial: Record<number, boolean> = {};
    instances.forEach((_, index) => {
      initial[index] = false;
    });
    return initial;
  });

  const formatDate = (dateStr: string): string => {
    if (!dateStr) return "Unknown";
    const match = dateStr.match(/(\d+)\/(\d+)\/(\d+)\s*-\s*(.+)/);
    if (!match) return dateStr;

    const [, month, day, year, time] = match;
    return `${month}/${day}/${year} - ${time}`;
  };

  const handleSelectionChange = (index: number) => {
    const newSelection = new Set(selectedIndices);
    if (newSelection.has(index)) {
      newSelection.delete(index);
    } else {
      newSelection.add(index);
    }
    setSelectedIndices(newSelection);
  };

  const handleSelectAllChange = () => {
    if (selectedIndices.size === instances.length) {
      setSelectedIndices(new Set());
    } else {
      const allIndices = new Set(instances.map((_, index) => index));
      setSelectedIndices(allIndices);
    }
  };

  const handleProcessClick = () => {
    const selected = instances
      .map((instance, index) => ({ instance, index }))
      .filter(({ index }) => selectedIndices.has(index))
      .map(({ instance, index }) => ({
        ...instance,
        serverName: instanceServerSelection[index] || selectedServer,
        serverVerified: !!instanceServerVerified[index],
      }));
    if (selected.length > 0) {
      onProcess(selected);
    }
  };

  const handleInstanceServerChange = (index: number, value: string) => {
    setInstanceServerSelection((prev) => ({ ...prev, [index]: value }));
    setInstanceServerVerified((prev) => ({ ...prev, [index]: false }));
  };

  const handleInstanceVerificationChange = (index: number, checked: boolean) => {
    setInstanceServerVerified((prev) => ({ ...prev, [index]: checked }));
  };

  const allSelected =
    selectedIndices.size === instances.length && instances.length > 0;
  const someSelected =
    selectedIndices.size > 0 && selectedIndices.size < instances.length;
  const hasUnverifiedSelected =
    hasMultipleDetectedServers &&
    Array.from(selectedIndices).some((idx) => !instanceServerVerified[idx]);
  const canProcess =
    !isProcessing && selectedIndices.size > 0 && !hasUnverifiedSelected;

  /** Map backend/internal server value to client-facing label (matches dropdown). */
  const serverLabel = (internalValue: string): string => {
    if (!internalValue) return "Unknown";
    const opt = serverOptions.find((o) => o.value === internalValue);
    return opt?.label ?? internalValue;
  };

  return (
    <div className="instance-selector">
      <h2>Select Raid Instances to Process</h2>

      <div className="select-all-container">
        <input
          type="checkbox"
          id="select-all"
          checked={allSelected}
          ref={(input) => {
            if (input) input.indeterminate = someSelected;
          }}
          onChange={handleSelectAllChange}
          disabled={isProcessing}
        />
        <label htmlFor="select-all">Select All Instances</label>
      </div>

      <div className="instance-list">
        {instances.map((instance, index) => {
          const bosses = instance.preview?.bosses || [];
          const hasBosses = bosses.length > 0;
          const firstBoss = hasBosses ? bosses[0] : "N/A";
          const lastBoss = hasBosses ? bosses[bosses.length - 1] : "N/A";

          return (
            <div key={index} className="instance-item">
              <input
                type="checkbox"
                id={`instance-${index}`}
                checked={selectedIndices.has(index)}
                onChange={() => handleSelectionChange(index)}
                disabled={isProcessing}
              />
              <div className="instance-content">
                <div className="instance-column">
                  <div className="info-row">
                    <span className="info-label">First Boss:</span>
                    <span className="info-value">{firstBoss}</span>
                  </div>
                  <div className="info-row">
                    <span className="info-label">Last Boss:</span>
                    <span className="info-value">{lastBoss}</span>
                  </div>
                </div>

                <div className="vertical-separator"></div>

                <div className="instance-column">
                  <div className="info-row">
                    <span className="info-label">Start:</span>
                    <span className="info-value">
                      {formatDate(instance.preview?.formattedStartTime || "")}
                    </span>
                  </div>
                  <div className="info-row">
                    <span className="info-label">End:</span>
                    <span className="info-value">
                      {formatDate(instance.preview?.formattedEndTime || "")}
                    </span>
                  </div>
                </div>

                <div className="vertical-separator"></div>

                <div className="instance-column">
                  <div className="info-row">
                    <span className="info-label">Logged by:</span>
                    <span className="info-value">
                      {instance.preview?.loggedBy || "Unknown"}
                    </span>
                  </div>
                  {hasMultipleDetectedServers ? (
                    <>
                      <div className="info-row">
                        <span className="info-label">Detected:</span>
                        <span className="info-value">
                          {serverLabel(
                            instance.preview?.detectedServerName || ""
                          )}
                          {instance.preview?.detectedGuidPrefix
                            ? ` (GUID ${instance.preview.detectedGuidPrefix})`
                            : ""}
                        </span>
                      </div>
                      <div className="instance-row-block">
                        <label
                          className="info-label"
                          htmlFor={`slice-server-${index}`}
                        >
                          Slice server:
                        </label>
                        <select
                          id={`slice-server-${index}`}
                          className="slice-server-select"
                          value={instanceServerSelection[index] || selectedServer}
                          onChange={(e) =>
                            handleInstanceServerChange(index, e.target.value)
                          }
                          disabled={isProcessing}
                        >
                          <option value="">Select server...</option>
                          {serverOptions.map((opt) => (
                            <option
                              key={`slice-${index}-${opt.value}-${opt.id ?? "na"}`}
                              value={opt.value}
                            >
                              {opt.label}
                            </option>
                          ))}
                        </select>
                      </div>
                      <div className="verify-row">
                        <input
                          type="checkbox"
                          id={`verify-slice-${index}`}
                          checked={!!instanceServerVerified[index]}
                          onChange={(e) =>
                            handleInstanceVerificationChange(
                              index,
                              e.target.checked
                            )
                          }
                          disabled={isProcessing}
                        />
                        <label htmlFor={`verify-slice-${index}`}>
                          I confirm this slice server is correct.
                        </label>
                      </div>
                    </>
                  ) : (
                    <div className="info-row">
                      <span className="info-label">Detected server:</span>
                      <span className="info-value">
                        {serverLabel(
                          instance.preview?.detectedServerName ||
                            selectedServer
                        )}
                      </span>
                    </div>
                  )}
                </div>
              </div>
            </div>
          );
        })}
      </div>

      <div className="button-group">
        <button
          className="btn"
          onClick={handleProcessClick}
          disabled={!canProcess}
          title={
            hasUnverifiedSelected
              ? "Verify server selection for each selected slice."
              : ""
          }
        >
          {isProcessing
            ? "Queuing Jobs..."
            : `Process ${selectedIndices.size} Selected Instance${
                selectedIndices.size !== 1 ? "s" : ""
              }`}
        </button>
        <button
          className="btn btn-secondary"
          onClick={onCancel}
          disabled={isProcessing}
        >
          Cancel
        </button>
      </div>
    </div>
  );
}

export default InstanceSelector;
