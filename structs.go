package main

type PreprocessResponse struct {
	Message                    string     `json:"message"`
	PreprocessID               int        `json:"preprocessId"`
	Instances                  []Instance `json:"instances"`
	AutoQueued                 bool       `json:"autoQueued"`
	HasMultipleDetectedServers bool       `json:"hasMultipleDetectedServers"`
	ViewLogURL                 string     `json:"viewLogURL"`
}

type Instance struct {
	Name               string           `json:"name"`
	EncounterStartTime string           `json:"encounterStartTime"`
	StartMs            int64            `json:"startMs"`
	EndMs              int64            `json:"endMs"`
	LineStart          int              `json:"lineStart"`
	LineEnd            int              `json:"lineEnd"`
	ServerName         string           `json:"serverName,omitempty"`
	ServerVerified     bool             `json:"serverVerified,omitempty"`
	Preview            *InstancePreview `json:"preview"`
}

type InstancePreview struct {
	LoggedBy           string   `json:"loggedBy"`
	Bosses             []string `json:"bosses"`
	FormattedStartTime string   `json:"formattedStartTime"`
	FormattedEndTime   string   `json:"formattedEndTime"`
	DetectedServerName string   `json:"detectedServerName,omitempty"`
	DetectedGuidPrefix string   `json:"detectedGuidPrefix,omitempty"`
}

type JobStatusResponse struct {
	TotalJobs     int         `json:"totalJobs"`
	JobsCompleted int         `json:"jobsCompleted"`
	Logs          []LogStatus `json:"logs"`
	ViewLogURL    string      `json:"viewLogURL"`
}

type LogStatus struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
}

type UploaderServer struct {
	ID    int    `json:"id"`
	Value string `json:"value"`
	Label string `json:"label"`
}

type UploaderServersResponse struct {
	Servers []UploaderServer `json:"servers"`
	Count   int              `json:"count"`
}
