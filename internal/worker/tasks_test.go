package worker

import (
	"testing"
	"time"
)

func TestTaskType(t *testing.T) {
	cases := map[string]string{
		"email":            TaskEmail,
		"app_notification": TaskAppNotification,
	}

	for kind, want := range cases {
		got, ok := taskType(kind)
		if !ok || got != want {
			t.Fatalf("taskType(%q) = %q/%v, want %q/true", kind, got, ok, want)
		}
	}

	if _, ok := taskType("unknown"); ok {
		t.Fatal("taskType(unknown) ok = true, want false")
	}
}

func TestBackoff(t *testing.T) {
	cases := map[int32]time.Duration{
		0:  time.Second,
		1:  2 * time.Second,
		5:  32 * time.Second,
		6:  time.Minute,
		12: time.Minute,
	}

	for attempts, want := range cases {
		if got := backoff(attempts); got != want {
			t.Fatalf("backoff(%d) = %v, want %v", attempts, got, want)
		}
	}
}
