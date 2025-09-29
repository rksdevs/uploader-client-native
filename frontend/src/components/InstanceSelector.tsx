import React, { useState } from "react";
import { Instance } from "../types";

interface InstanceSelectorProps {
  instances: Instance[];
  onProcess: (selectedInstances: Instance[]) => void;
  onCancel: () => void;
  isProcessing: boolean;
}

function InstanceSelector({
  instances,
  onProcess,
  onCancel,
  isProcessing,
}: InstanceSelectorProps) {
  const [selectedIndices, setSelectedIndices] = useState<Set<number>>(
    new Set()
  );

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
    const selected = instances.filter((_, index) => selectedIndices.has(index));
    if (selected.length > 0) {
      onProcess(selected);
    }
  };

  const allSelected =
    selectedIndices.size === instances.length && instances.length > 0;
  const someSelected =
    selectedIndices.size > 0 && selectedIndices.size < instances.length;

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
          disabled={isProcessing || selectedIndices.size === 0}
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
