package config

import (
	pbmodel "github.com/emortalmc/proto-specs/gen/go/model/badge"
	"github.com/spf13/viper"
	"strings"
)

type BadgeConfig struct {
	Badges map[string]*Badge
}

type Badge struct {
	Id       string
	Priority int
	Required bool

	FriendlyName string
	ChatString   string

	// HoverText is parsed on the client side as MiniMessage
	HoverText []string

	GuiItem *BadgeGuiItem

	AutomaticGrants *BadgeAutomaticGrants
}

func (b *Badge) GetFormattedHoverText() string {
	return strings.Join(b.HoverText, "\n")
}

func (b *Badge) ToProto() *pbmodel.Badge {
	return &pbmodel.Badge{
		Id:           b.Id,
		Priority:     int64(b.Priority),
		Required:     b.Required,
		FriendlyName: b.FriendlyName,
		ChatString:   b.ChatString,
		HoverText:    b.GetFormattedHoverText(),
		GuiItem:      b.GuiItem.ToProto(),
	}
}

type BadgeGuiItem struct {
	// Display whether the item should be displayed in the GUI if not unlocked
	Display     bool
	Material    string
	DisplayName string
	Lore        []string
}

func (b *BadgeGuiItem) ToProto() *pbmodel.Badge_GuiItem {
	return &pbmodel.Badge_GuiItem{
		Material:    b.Material,
		DisplayName: b.DisplayName,
		Lore:        b.Lore,
	}
}

type BadgeAutomaticGrants struct {
	GitHubPullRequests *int // todo
	PermissionRole     *string
}

func LoadBadgeConfig() (config BadgeConfig, err error) {
	v := viper.New()
	v.AddConfigPath("./badge-config")
	v.SetConfigName("config")

	err = v.ReadInConfig()
	if err != nil {
		return
	}

	err = v.Unmarshal(&config)
	if err != nil {
		return
	}

	return
}
