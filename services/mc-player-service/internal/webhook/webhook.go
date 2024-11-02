package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"net/http"
)

type webhookImpl struct {
	discordWebhookUrl string
	log               *zap.SugaredLogger
	client            *http.Client
}

func NewWebhook(discordWebhookUrl string, log *zap.SugaredLogger) Webhook {
	w := &webhookImpl{
		discordWebhookUrl: discordWebhookUrl,
		log:               log,
		client:            &http.Client{},
	}

	if w.discordWebhookUrl == "" {
		w.log.Warn("discord webhook url is not present, webhook messages will not be sent")
	}

	return w
}

type jsonData struct {
	Username  string `json:"username"`
	Content   string `json:"content"`
	AvatarUrl string `json:"avatar_url"`
}

func (w *webhookImpl) sendWebhookMessage(payload []byte) {
	if w.discordWebhookUrl == "" {
		w.log.Debugw("discord webhook url is nil, not sending webhook message")
		return
	}

	req, err := http.NewRequest("POST", w.discordWebhookUrl, bytes.NewBuffer(payload))
	if err != nil {
		w.log.Errorw("error on http post", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Go-Discord")

	resp, err := w.client.Do(req)
	if err != nil {
		w.log.Errorw("error on request", err)
		return
	}

	if resp.StatusCode != 204 {
		w.log.Errorw("error on request", "status", resp.StatusCode, "url", w.discordWebhookUrl)
		return
	}

	if err := resp.Body.Close(); err != nil {
		w.log.Errorw("error closing body", err)
		return
	}
}

func (w *webhookImpl) SendPlayerJoinWebhook(username string, uuid string, plrCount int64) {
	playersText := "players"
	if plrCount == 1 {
		playersText = "player"
	}

	jsonData, err := json.Marshal(jsonData{
		username,
		fmt.Sprintf("Joined the server! (%d %s)", plrCount, playersText),
		fmt.Sprintf("https://mc-heads.net/avatar/%s/100", uuid),
	})
	if err != nil {
		w.log.Errorw("error marshalling json", err)
		return
	}

	go w.sendWebhookMessage(jsonData)
}

func (w *webhookImpl) SendPlayerLeaveWebhook(username string, uuid string, plrCount int64) {
	playersText := "players"
	if plrCount == 1 {
		playersText = "player"
	}

	jsonData, err := json.Marshal(jsonData{
		username,
		fmt.Sprintf("Left the server! (%d %s)", plrCount, playersText),
		fmt.Sprintf("https://mc-heads.net/avatar/%s/100", uuid),
	})
	if err != nil {
		w.log.Errorw("error marshalling json", err)
		return
	}

	go w.sendWebhookMessage(jsonData)
}
