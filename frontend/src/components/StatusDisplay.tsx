import React from "react";

interface StatusDisplayProps {
  message: string;
}

const StatusDisplay: React.FC<StatusDisplayProps> = ({ message }) => {
  return (
    <div className="status-container">
      <p>{message}</p>
    </div>
  );
};

export default StatusDisplay;
