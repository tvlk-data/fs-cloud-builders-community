package slackbot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	cloudbuild "google.golang.org/api/cloudbuild/v1"
)

func NewWebhookNotifier(webhook string, project string, projectId string, alertLevel AlertLevel) *WebhookNotifier {
	return &WebhookNotifier{
		webhook, project, projectId, alertLevel,
	}
}

func (n *WebhookNotifier) Notify(build *cloudbuild.Build) {
	url := fmt.Sprintf("https://console.cloud.google.com/cloud-build/builds/%s?project=%s", build.Id, n.projectId)
	var emoji string
	switch build.Status {
	case "WORKING":
		emoji = ":hammer:"
	case "SUCCESS":
		emoji = ":white_check_mark:"
	case "FAILURE":
		emoji = ":x:"
	case "CANCELLED":
		emoji = ":wastebasket:"
	case "TIMEOUT":
		emoji = ":hourglass:"
	case "STATUS_UNKNOWN", "INTERNAL_ERROR":
		emoji = ":interrobang:"
	default:
		emoji = ":question:"
	}
	tagAlert := ""
	if n.alertLevel == AlertLevel_ERR && (build.Status == "TIMEOUT" || build.Status == "FAILURE" || build.Status == "INTERNAL_ERROR") {
		tagAlert = "<!subteam^S014F8S6EBH> "
	}

	// Ensure messages remain the same as before
	if n.project == "unknown" {
		n.project = ""
	}

	var msg string
	if build.Status == "WORKING" {
		msgFmt := `{
			"text": "%s *%s* build started",
			"attachments": [{
				"fallback": "Open build details at %s",
				"actions": [{
					"type": "button",
					"text": "Open details",
					"url": "%s"
				}]
			}]
		}`
		msg = fmt.Sprintf(msgFmt, emoji, n.project, url, url)
	} else {
		startTime, err := time.Parse(time.RFC3339, build.StartTime)
		if err != nil {
			log.Fatalf("Failed to parse Build.StartTime: %v", err)
		}
		finishTime, err := time.Parse(time.RFC3339, build.FinishTime)
		if err != nil {
			log.Fatalf("Failed to parse Build.FinishTime: %v", err)
		}
		buildDuration := finishTime.Sub(startTime).Truncate(time.Second)

		msgFmt := `{
			"text": "%s *%s* build _%s_ after _%s_ %s",
			"attachments": [{
				"fallback": "Open build details at %s",
				"actions": [{
					"type": "button",
					"text": "Open details",
					"url": "%s"
				}]
			}]
		}`
		msg = fmt.Sprintf(msgFmt, emoji, n.project, build.Status, buildDuration, tagAlert, url, url)
	}

	r := strings.NewReader(msg)
	resp, err := http.Post(n.webhook, "application/json", r)
	if err != nil {
		log.Fatalf("Failed to post to Slack: %v", err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	log.Printf("Posted message to Slack: [%v], got response [%s]", msg, body)
}

func (n *WebhookNotifier) NotifyStep(build *cloudbuild.Build) {

}

func NewTokenNotifier(project string, projectId string, channel string, token string, alertLevel AlertLevel) *TokenNotifier {
	return &TokenNotifier{
		project, projectId, channel, token, alertLevel, "", -1,
	}
}

func (n *TokenNotifier) Notify(build *cloudbuild.Build) {
	url := fmt.Sprintf("https://console.cloud.google.com/cloud-build/builds/%s?project=%s", build.Id, n.projectId)
	var emoji string

	switch build.Status {
	case "WORKING":
		emoji = ":hammer:"
	case "SUCCESS":
		emoji = ":white_check_mark:"
	case "FAILURE":
		emoji = ":x:"
	case "CANCELLED":
		emoji = ":wastebasket:"
	case "TIMEOUT":
		emoji = ":hourglass:"
	case "STATUS_UNKNOWN", "INTERNAL_ERROR":
		emoji = ":interrobang:"
	default:
		emoji = ":question:"
	}

	tagAlert := ""
	if n.alertLevel == AlertLevel_ERR && (build.Status == "TIMEOUT" || build.Status == "FAILURE" || build.Status == "INTERNAL_ERROR") {
		tagAlert = "<!subteam^S014F8S6EBH> "
	}
	// Ensure messages remain the same as before
	if n.project == "unknown" {
		n.project = ""
	}

	var msg string
	if build.Status == "WORKING" {
		msgFmt := `{
			"text": "%s *%s* build started",
			"attachments": [{
				"fallback": "Open build details at %s",
				"actions": [{
					"type": "button",
					"text": "Open details",
					"url": "%s"
				}]
			}],
			"channel": "%s"
		}`
		msg = fmt.Sprintf(msgFmt, emoji, n.project, url, url, n.channel)
	} else {
		startTime, err := time.Parse(time.RFC3339, build.StartTime)
		if err != nil {
			log.Fatalf("Failed to parse Build.StartTime: %v", err)
		}
		finishTime, err := time.Parse(time.RFC3339, build.FinishTime)
		if err != nil {
			log.Fatalf("Failed to parse Build.FinishTime: %v", err)
		}
		buildDuration := finishTime.Sub(startTime).Truncate(time.Second)

		var msgFmt string
		msgFmt = `{
			"text": "%s *%s* build _%s_ after _%s_ %s",
			"attachments": [{
				"fallback": "Open build details at %s",
				"actions": [{
					"type": "button",
					"text": "Open details",
					"url": "%s"
				}]
			}],
			"channel": "%s",
			"thread_ts": "%s"
		}`
		msg = fmt.Sprintf(msgFmt, emoji, n.project, build.Status, buildDuration, tagAlert, url, url, n.channel, n.threadTs)
	}

	client := &http.Client{}
	r, err := http.NewRequest(http.MethodPost, "https://slack.com/api/chat.postMessage", strings.NewReader(msg))
	if err != nil {
		log.Fatalf("Unable to construct http call, %s", err.Error())
	}
	r.Header.Add("Authorization", "Bearer "+n.token)
	r.Header.Add("Content-Type", "application/json")

	response, err := client.Do(r)
	if err != nil {
		log.Fatalf("Unable to send Slack message, %s", err.Error())
	}

	defer response.Body.Close()
	var responseBody SlackCreateMessageResponse
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("Unable to decode Slack response, %s", err.Error())
	}
	err = json.Unmarshal(body, &responseBody)
	if err != nil {
		log.Fatalf("Unable to unmarshal JSON Slack response, %s", err.Error())
	}
	log.Printf("Posted message to Slack: [%v], got response [%s], with thread ts [%s]", msg, responseBody.Ok, responseBody.Ts)

	if n.threadTs == "" {
		n.threadTs = responseBody.Ts
	}
}

func (n *TokenNotifier) NotifyStep(build *cloudbuild.Build) {
	for i, step := range build.Steps {
		var emoji string
		log.Printf("Status %s, with last step %d", step.Status, n.lastStepNotified)
		switch step.Status {
		case "SUCCESS":
			emoji = ":check_green:"
		case "FAILURE", "INTERNAL_ERROR", "TIMEOUT", "EXPIRED":
			emoji = ":x:"
		case "STATUS_UNKNOWN", "CANCELLED":
			emoji = ":notsureif:"
		case "QUEUED", "WORKING", "":
			return
		}

		msgFmt := `{
			"text": "%s *Step %d*: %s",
			"thread_ts": "%s",
			"channel": "%s"
		}`
		msg := fmt.Sprintf(msgFmt, emoji, i, step.Id, n.threadTs, n.channel)

		client := &http.Client{}
		r, err := http.NewRequest(http.MethodPost, "https://slack.com/api/chat.postMessage", strings.NewReader(msg))
		if err != nil {
			log.Fatalf("Unable to construct http call, %s", err.Error())
		}
		r.Header.Add("Authorization", "Bearer "+n.token)
		r.Header.Add("Content-Type", "application/json")

		response, err := client.Do(r)
		if err != nil {
			log.Fatalf("Unable to send Slack message, %s", err.Error())
		}

		defer response.Body.Close()
		var responseBody SlackCreateMessageResponse
		body, err := ioutil.ReadAll(response.Body)

		if err != nil {
			log.Fatalf("Unable to decode Slack response, %s", err.Error())
		}
		err = json.Unmarshal(body, &responseBody)
		if err != nil {
			log.Fatalf("Unable to unmarshal JSON Slack response, %s", err.Error())
		}
		log.Printf("Posted message to Slack: [%v], got response [%s], with thread ts [%s]", msg, responseBody.Ok, responseBody.Ts)

		n.lastStepNotified = i
	}

}
