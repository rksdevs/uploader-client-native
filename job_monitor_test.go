package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClassifyJobOutcome(t *testing.T) {
	cases := []struct {
		name             string
		uploaded, failed int
		timedOut         bool
		want             jobOutcome
	}{
		{"all uploaded", 3, 0, false, jobOutcomeAllUploaded},
		{"some failed", 2, 1, false, jobOutcomePartial},
		{"all failed", 0, 2, false, jobOutcomeAllFailed},
		{"timed out with nothing back is not a failure", 0, 0, true, jobOutcomeTimedOut},
		{"timed out wins over partial results", 1, 1, true, jobOutcomeTimedOut},
	}
	for _, c := range cases {
		if got := classifyJobOutcome(c.uploaded, c.failed, c.timedOut); got != c.want {
			t.Errorf("%s: got %d want %d", c.name, got, c.want)
		}
	}
}

func TestJobPollIntervalBacksOff(t *testing.T) {
	if jobPollInterval(time.Minute) != jobPollFast {
		t.Fatal("expected fast polling at the start")
	}
	if jobPollInterval(jobPollFastWindow) != jobPollSlow {
		t.Fatal("expected slow polling after the fast window")
	}
}

func TestCheckJobStatusReadsQueuePositionAndSendsHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(nativeUploaderClientHeader) != "1" {
			t.Errorf("missing native header")
		}
		_, _ = w.Write([]byte(`{"totalJobs":2,"jobsCompleted":0,"queuePosition":4,"logs":[]}`))
	}))
	defer server.Close()

	app := &App{apiBaseURL: server.URL}
	status, err := app.checkJobStatus(12)
	if err != nil {
		t.Fatal(err)
	}
	if status.QueuePosition == nil || *status.QueuePosition != 4 {
		t.Fatalf("queue position: %+v", status.QueuePosition)
	}
}
