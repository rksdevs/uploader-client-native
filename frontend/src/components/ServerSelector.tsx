import React from "react";

interface ServerSelectorProps {
  selectedValue: string;
  onSelect: (value: string) => void;
  disabled: boolean;
}

const ServerSelector: React.FC<ServerSelectorProps> = ({
  selectedValue,
  onSelect,
  disabled,
}) => {
  const serverOptions = [
    { label: "Select a Server...", value: "" },
    { label: "Whitemane-Frostmourne", value: "Whitemane_Frostmourne" },
    { label: "Warmane-Icecrown", value: "Warmane_Icecrown" },
    { label: "Warmane-Onyxia", value: "Warmane_Onyxia" },
    { label: "Sunwell", value: "Sunwell" },
    { label: "AstraWow-Wrathion", value: "AstraWow_Wrathion" },
    { label: "AstraWow-Neltharion", value: "AstraWow_Neltharion" },
    { label: "Warmane-Lordaeron", value: "Warmane_Lordaeron" },
  ];

  return (
    <div className="component-container">
      <label htmlFor="server-select">2. Choose Your Server</label>
      <select
        id="server-select"
        value={selectedValue}
        onChange={(e) => onSelect(e.target.value)}
        disabled={disabled}
      >
        {serverOptions.map((opt) => (
          <option key={opt.value} value={opt.value}>
            {opt.label}
          </option>
        ))}
      </select>
    </div>
  );
};

export default ServerSelector;
