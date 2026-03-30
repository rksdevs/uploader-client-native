export interface InstancePreview {
  loggedBy: string;
  bosses: string[];
  formattedStartTime: string;
  formattedEndTime: string;
  detectedServerName?: string;
  detectedGuidPrefix?: string;
}

export interface Instance {
  name: string;
  encounterStartTime: string;
  startMs: number;
  endMs: number;
  lineStart: number;
  lineEnd: number;
  serverName?: string;
  serverVerified?: boolean;
  preview?: InstancePreview;
}

export interface PreprocessResponse {
  message: string;
  preprocessId: number;
  instances: Instance[];
  autoQueued: boolean;
  hasMultipleDetectedServers: boolean;
  viewLogURL: string;
}
export interface JobStatusResponse {
  totalJobs: number;
  jobsCompleted: number;
  logs: LogStatus[];
  viewLogURL: string;
}

export interface LogStatus {
  id: number;
  status: string;
}

export interface JobNotification {
  logId: number;
  status: "uploaded" | "failed";
  error?: string;
  viewLogURL: string;
}