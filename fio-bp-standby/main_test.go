package main

import (
	"os"
	"testing"
	"time"
)

func TestNotify(t *testing.T) {
	key := os.Getenv("KEY")
	if key == "" {
		t.SkipNow()
	}

	err := notifyPagerduty(false, "testing trigger", "test-alert", key, "go test")
	if err != nil {
		t.Error(err)
	}

	time.Sleep(3*time.Second)

	err = notifyPagerduty(true, "testing resolve", "test-alert", key, "go test")
	if err != nil {
		t.Error(err)
	}

}
