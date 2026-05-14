import React from "react";
import { FolderOpen } from "lucide-react";

interface DirectorySelectorProps {
  onSelect: () => void;
  disabled: boolean;
}

const DirectorySelector: React.FC<DirectorySelectorProps> = ({
  onSelect,
  disabled,
}) => {
  return (
    <button
      type="button"
      onClick={onSelect}
      className="btn-gradient-primary btn-with-icon"
      disabled={disabled}
    >
      <FolderOpen size={18} strokeWidth={2} aria-hidden />
      Choose Logs directory
    </button>
  );
};

export default DirectorySelector;
