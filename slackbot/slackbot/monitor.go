package slackbot

import (
	"context"
	"log"
	"time"
)

const maxErrors = 3
const tickDuration = 20 * time.Second

// 1m20s
const monitorErrorMarginDuration = (maxErrors + 1) * tickDuration

// Monitor polls Cloud Build until the build reaches completed status, then triggers the Slack event.
func Monitor(ctx context.Context, projectId string, buildId string, webhook string, project string, auth string, channel string, env string, token string) {
	svc := gcbClient(ctx)
	errors := 0
	started := false
	var notifier Notifier
	var alertLevel AlertLevel
	if env == "dev" {
		alertLevel = AlertLevel_INFO
	} else {
		alertLevel = AlertLevel_ERR
	}
	if auth == "token" {
		notifier = NewTokenNotifier(project, projectId, channel, token, alertLevel)
	} else if auth == "webhook" {
		notifier = NewWebhookNotifier(webhook, project, projectId, alertLevel)
	}
	t := time.Tick(tickDuration)
	for {
		log.Printf("Polling build %s", buildId)
		getMonitoredBuild := svc.Projects.Builds.Get(projectId, buildId)
		monitoredBuild, err := getMonitoredBuild.Do()
		if err != nil {
			if errors <= maxErrors {
				log.Printf("Failed to get build details from Cloud Build.  Will retry in %s", tickDuration)
				errors++
				continue
			} else {
				log.Fatalf("Reached maximum number of errors (%d).  Exiting", maxErrors)
			}
		}
		switch monitoredBuild.Status {
		case "WORKING":
			if !started {
				log.Printf("Build started. Notifying")
				notifier.Notify(monitoredBuild)
				started = true
			} else {
				notifier.NotifyStep(monitoredBuild)
			}
		case "SUCCESS", "FAILURE", "INTERNAL_ERROR", "TIMEOUT", "CANCELLED":
			log.Printf("Terminal status reached. Notifying")
			notifier.NotifyStep(monitoredBuild)
			notifier.Notify(monitoredBuild)
			return
		}
		<-t
	}
}
