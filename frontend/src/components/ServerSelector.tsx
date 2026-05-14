import React from "react";
import { ChevronDown } from "lucide-react";

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
    <div className="select-like-wrap">
      <select
        id="server-select"
        className="input-like-select"
        value={selectedValue}
        onChange={(e) => onSelect(e.target.value)}
        disabled={disabled}
      >
        <option value="">Select a server…</option>
        {serverOptions.map((opt) => (
          <option key={`${opt.value}-${opt.id ?? "na"}`} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
      <ChevronDown
        className="select-like-wrap__chevron"
        size={20}
        strokeWidth={2}
        aria-hidden
      />
    </div>
  );
};

export default ServerSelector;
