package webhook

type Webhook interface {
	SendPlayerJoinWebhook(username string, uuid string, plrCount int64)
	SendPlayerLeaveWebhook(username string, uuid string, plrCount int64)
}
