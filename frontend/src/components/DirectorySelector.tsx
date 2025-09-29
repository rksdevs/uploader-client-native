import React from "react";

interface DirectorySelectorProps {
  onSelect: () => void;
  disabled: boolean;
}

const DirectorySelector: React.FC<DirectorySelectorProps> = ({
  onSelect,
  disabled,
}) => {
  return (
    <div className="component-container">
      <label>1. Select WoW Logs Folder</label>
      <button onClick={onSelect} className="btn" disabled={disabled}>
        Choose Directory
      </button>
    </div>
  );
};

export default DirectorySelector;
