import React from "react";

export interface ServerOption {
  id?: number;
  value: string;
  label: string;
}

interface ServerSelectorProps {
  selectedValue: string;
  onSelect: (value: string) => void;
  disabled: boolean;
  serverOptions: ServerOption[];
}

const ServerSelector: React.FC<ServerSelectorProps> = ({
  selectedValue,
  onSelect,
  disabled,
  serverOptions,
}) => {
  return (
    <div className="component-container">
      <label htmlFor="server-select">2. Choose Your Server</label>
      <select
        id="server-select"
        value={selectedValue}
        onChange={(e) => onSelect(e.target.value)}
        disabled={disabled}
      >
        <option value="">Select a Server...</option>
        {serverOptions.map((opt) => (
          <option key={`${opt.value}-${opt.id ?? "na"}`} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
    </div>
  );
};

export default ServerSelector;
