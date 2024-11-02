package kafka

import (
	"bytes"
	"context"
	"fmt"
	"github.com/emortalmc/proto-specs/gen/go/grpc/badge"
	"github.com/emortalmc/proto-specs/gen/go/grpc/permission"
	"github.com/emortalmc/proto-specs/gen/go/message/common"
	permmsg "github.com/emortalmc/proto-specs/gen/go/message/permission"
	badgepbmodel "github.com/emortalmc/proto-specs/gen/go/model/badge"
	pbmodel "github.com/emortalmc/proto-specs/gen/go/model/messagehandler"
	permmodel "github.com/emortalmc/proto-specs/gen/go/model/permission"
	"github.com/emortalmc/proto-specs/gen/go/nongenerated/kafkautils"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"message-handler/internal/config"
	"sync"
	"text/template"
	"time"
)

const consumeMessagesTopic = "mc-messages"
const consumePermissionsTopic = "permission-manager"

type consumer struct {
	logger *zap.SugaredLogger

	notifier Notifier

	permClient  permission.PermissionServiceClient
	badgeClient badge.BadgeManagerClient

	roleCache map[string]*permmodel.Role
}

func NewConsumer(ctx context.Context, wg *sync.WaitGroup, cfg *config.KafkaConfig, logger *zap.SugaredLogger, notifier Notifier,
	permClient permission.PermissionServiceClient, badgeClient badge.BadgeManagerClient) {

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)},
		GroupID:     "message-handler-service",
		GroupTopics: []string{consumeMessagesTopic, consumePermissionsTopic},

		Logger: kafka.LoggerFunc(func(format string, args ...interface{}) {
			logger.Infow(fmt.Sprintf(format, args...))
		}),
		ErrorLogger: kafka.LoggerFunc(func(format string, args ...interface{}) {
			logger.Errorw(fmt.Sprintf(format, args...))
		}),

		MaxWait: 5 * time.Second,
	})

	c := &consumer{
		logger: logger,

		notifier: notifier,

		permClient:  permClient,
		badgeClient: badgeClient,

		roleCache: make(map[string]*permmodel.Role),
	}

	c.cacheRoles(ctx)
	logger.Infow("cached roles", "count", len(c.roleCache))

	handler := kafkautils.NewConsumerHandler(logger, reader)
	handler.RegisterHandler(&common.PlayerChatMessageMessage{}, c.handlePlayerChatMessage)
	handler.RegisterHandler(&permmsg.RoleUpdateMessage{}, c.handleRoleUpdate)

	logger.Infow("starting listening for kafka messages", "topics", reader.Config().GroupTopics)

	wg.Add(1)
	go func() {
		defer wg.Done()
		handler.Run(ctx) // Run is blocking until the context is cancelled
		if err := reader.Close(); err != nil {
			logger.Errorw("error closing kafka reader", "error", err)
		}
	}()
}

func (c *consumer) cacheRoles(ctx context.Context) {
	res, err := c.permClient.GetAllRoles(ctx, &permission.GetAllRolesRequest{})
	if err != nil {
		c.logger.Panicf("failed to get all roles: %v", err)
	}

	for _, role := range res.Roles {
		c.roleCache[role.Id] = role
	}
}

var chatTemplate = template.Must(template.New("chat").Parse("{{if .Badge}}<hover:show_text:'{{.BadgeHoverDescription}}'>{{.Badge}}</hover> {{end}}{{.DisplayName}}: <content>"))

type chatTemplateData struct {
	Badge                 string
	BadgeHoverDescription string

	DisplayName string
}

// todo in the future let's make this resistant to service failures
func (c *consumer) handlePlayerChatMessage(ctx context.Context, _ *kafka.Message, uncastMsg proto.Message) {
	msg := uncastMsg.(*common.PlayerChatMessageMessage)

	originalMessage := msg.Message

	// 1. Get active badge
	// 2. Get active prefix
	// 3. Get active username
	// Process in chat template :D
	b, err := c.fetchPlayerBadge(ctx, originalMessage.SenderId)
	if err != nil {
		c.logger.Errorw("failed to get player b", err) // Log but continue
	}

	rolesResp, err := c.fetchPlayerRoles(ctx, originalMessage.SenderId)
	if err != nil {
		c.logger.Errorw("failed to get player roles", err)
		return
	}

	roles, err := c.roleIdsToRoles(rolesResp.RoleIds)
	if err != nil {
		c.logger.Errorw("failed to get player roles", err)
		return
	}

	displayNamePart, err := c.getDisplayUsername(originalMessage.SenderUsername, rolesResp)
	if err != nil {
		c.logger.Errorw("failed to get player displayNamePart", err) // Log but continue
	}

	messageContent := sanitizeMessage(originalMessage.Message)

	templateData := &chatTemplateData{
		DisplayName: displayNamePart,
	}

	if b != nil {
		templateData.Badge = b.ChatString
		templateData.BadgeHoverDescription = b.HoverText
	}

	message, err := createMessage(templateData)

	if err != nil {
		c.logger.Errorw("failed to create chat message", err)
		return
	}

	if err := c.notifier.ChatMessageCreated(ctx, &pbmodel.ChatMessage{
		SenderId:       originalMessage.SenderId,
		SenderUsername: originalMessage.SenderUsername,

		Message:        message,
		MessageContent: messageContent,

		ParseMessageContent: playerHasPermission(roles, "chat.parse"),
	}); err != nil {
		c.logger.Errorw("failed to notify chat message created", err)
	}
}

func (c *consumer) roleIdsToRoles(roleIds []string) ([]*permmodel.Role, error) {
	roles := make([]*permmodel.Role, len(roleIds))

	for i, roleId := range roleIds {
		role, ok := c.roleCache[roleId]
		if !ok {
			return nil, fmt.Errorf("failed to find role with id %s", roleId)
		}

		roles[i] = role
	}

	return roles, nil
}

func playerHasPermission(roles []*permmodel.Role, perm string) bool {
	// todo this is technically wrong as it doesn't use priority levels
	for _, role := range roles {
		for _, p := range role.Permissions {
			if p.State == permmodel.PermissionNode_ALLOW && p.Node == perm {
				return true
			}
		}
	}

	return false
}

const (
	minBlockedRune = '\uE000'
	maxBlockedRune = '\uF8FF'
)

func sanitizeMessage(message string) string {
	runes := []rune(message)
	for i, r := range runes {
		if r >= minBlockedRune && r <= maxBlockedRune {
			runes[i] = ' '
		}
	}

	return string(runes)
}

func createMessage(data *chatTemplateData) (string, error) {
	var buf bytes.Buffer
	if err := chatTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute chat template: %w", err)
	}
	return buf.String(), nil
}

// fetchPlayerBadge returns a string (including a space if a badge is present) representing the player's active badge.
// If no badge is present, an empty string is returned.
// If there is an error, the string "?? " is returned.
func (c *consumer) fetchPlayerBadge(ctx context.Context, playerId string) (*badgepbmodel.Badge, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	res, err := c.badgeClient.GetActivePlayerBadge(ctx, &badge.GetActivePlayerBadgeRequest{
		PlayerId: playerId,
	})
	if err != nil {
		if s, ok := status.FromError(err); ok {
			if s.Code() == codes.NotFound {
				return nil, nil
			}
			return nil, fmt.Errorf("failed to get player badge (status: %s): %w", s.Code(), err)
		}
		return nil, fmt.Errorf("failed to get player badge (status: unknown): %w", err)
	}

	if res.Badge == nil {
		return nil, nil
	}

	return res.Badge, nil
}

func (c *consumer) getDisplayUsername(username string, rolesResp *permission.PlayerRolesResponse) (string, error) {
	if rolesResp.ActiveDisplayNameRoleId == nil {
		return username, nil
	}

	if rolesResp.ActiveDisplayNameRoleId == nil {
		return username, nil
	}

	role, ok := c.roleCache[*rolesResp.ActiveDisplayNameRoleId]
	if !ok {
		return username, fmt.Errorf("failed to find role with id %s", *rolesResp.ActiveDisplayNameRoleId)
	}
	if role.DisplayName == nil {
		return username, fmt.Errorf("role with id %s has no display name (but should have?)", *rolesResp.ActiveDisplayNameRoleId)
	}

	t, err := template.New("displayname").Parse(*role.DisplayName)
	if err != nil {
		return username, fmt.Errorf("failed to parse display name template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, struct{ Username string }{Username: username}); err != nil {
		return username, fmt.Errorf("failed to execute display name template: %w", err)
	}

	return buf.String(), nil
}

func (c *consumer) fetchPlayerRoles(ctx context.Context, playerId string) (*permission.PlayerRolesResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	res, err := c.permClient.GetPlayerRoles(ctx, &permission.GetPlayerRolesRequest{
		PlayerId: playerId,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get player roles: %w", err)
	}

	return res, nil
}

func (c *consumer) handleRoleUpdate(_ context.Context, _ *kafka.Message, uncastMsg proto.Message) {
	msg := uncastMsg.(*permmsg.RoleUpdateMessage)

	switch msg.ChangeType {
	case permmsg.RoleUpdateMessage_CREATE, permmsg.RoleUpdateMessage_MODIFY:
		c.roleCache[msg.Role.Id] = msg.Role
	case permmsg.RoleUpdateMessage_DELETE:
		delete(c.roleCache, msg.Role.Id)
	}
}
