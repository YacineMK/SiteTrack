package worker

import (
	"fmt"
	"os"
)

const (
	QueueName           = "notifications"
	TaskEmail           = "notifications:email"
	TaskAppNotification = "notifications:app"
)

func taskType(kind string) (string, bool) {
	switch kind {
	case "email":
		return TaskEmail, true
	case "app_notification":
		return TaskAppNotification, true
	default:
		return "", false
	}
}

func workerID() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "worker"
	}
	return fmt.Sprintf("%s-%d", host, os.Getpid())
}
