import { main } from "../wailsjs/go/models";

export type Instance = main.Instance;
export type InstancePreview = main.InstancePreview;

export interface PreprocessResponse {
  message: string;
  preprocessId: number;
  instances: Instance[];
  autoQueued: boolean;
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