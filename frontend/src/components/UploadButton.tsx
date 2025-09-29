import React from "react";

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
      onClick={onUpload}
      disabled={disabled}
      className={`upload-button ${isProcessing ? "processing" : ""}`}
    >
      {isProcessing ? "Processing..." : "Upload & Process Log"}
    </button>
  );
};

export default UploadButton;
