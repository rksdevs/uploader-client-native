import React from "react";
import { Upload } from "lucide-react";

interface UploadButtonProps {
  onUpload: () => void;
  disabled: boolean;
  isProcessing: boolean;
}

const UploadButton: React.FC<UploadButtonProps> = ({
  onUpload,
  disabled,
  isProcessing,
}) => {
  return (
    <button
      type="button"
      onClick={onUpload}
      disabled={disabled}
      className={`btn-gradient-primary btn-with-icon upload-cta ${isProcessing ? "processing" : ""}`}
    >
      <Upload size={18} strokeWidth={2} aria-hidden />
      {isProcessing ? "Processing…" : "Upload & Process Log"}
    </button>
  );
};

export default UploadButton;
