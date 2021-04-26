package slackbot

import (
	cloudbuild "google.golang.org/api/cloudbuild/v1"
)

type SlackCreateMessageResponse struct {
	Ok      bool   `json:"ok"`
	Channel string `json:"channel"`
	Ts      string `json:"ts"`
}

type AlertLevel int32

const (
	AlertLevel_INFO AlertLevel = 0
	AlertLevel_ERR  AlertLevel = 1
)

type Notifier interface {
	Notify(build *cloudbuild.Build)
	NotifyStep(build *cloudbuild.Build)
}

type WebhookNotifier struct {
	webhook    string
	project    string
	projectId  string
	alertLevel AlertLevel
}

type TokenNotifier struct {
	project          string
	projectId        string
	channel          string
	token            string
	alertLevel       AlertLevel
	threadTs         string
	lastStepNotified int
}
