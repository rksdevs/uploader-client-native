package main

type PreprocessResponse struct {
	Message      string     `json:"message"`
	PreprocessID int        `json:"preprocessId"`
	Instances    []Instance `json:"instances"`
	AutoQueued   bool       `json:"autoQueued"`
	ViewLogURL   string     `json:"viewLogURL"`
}

type Instance struct {
	Name               string           `json:"name"`
	EncounterStartTime string           `json:"encounterStartTime"`
	StartMs            int64            `json:"startMs"`
	EndMs              int64            `json:"endMs"`
	LineStart          int              `json:"lineStart"`
	LineEnd            int              `json:"lineEnd"`
	Preview            *InstancePreview `json:"preview"`
}

type InstancePreview struct {
	LoggedBy           string   `json:"loggedBy"`
	Bosses             []string `json:"bosses"`
	FormattedStartTime string   `json:"formattedStartTime"`
	FormattedEndTime   string   `json:"formattedEndTime"`
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
