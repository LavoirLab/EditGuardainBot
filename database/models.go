package database

import "time"

type EchoSettings struct {
	Mode  string `bson:"mode"`
	Limit int    `bson:"limit"`
}

type EditModeSettings struct {
	Enabled  bool          `bson:"enabled"`
	Duration time.Duration `bson:"duration"`
}

type LinkData struct {
	Hostnames []string `bson:"hostnames"`
	Enabled   bool     `bson:"enabled"`
}

type MediaDeleteSettings struct {
	Enabled bool          `bson:"enabled"`
	Delay   time.Duration `bson:"delay"`
}

type MsgDeleteSettings struct {
	Enabled bool          `bson:"enabled"`
	Delay   time.Duration `bson:"delay"`
}

type FileDeleteSettings struct {
	Enabled bool          `bson:"enabled"`
	Delay   time.Duration `bson:"delay"`
}

type ChatSettings struct {
	ChatID            int64               `bson:"_id"`
	EditMode          EditModeSettings    `bson:"edit_mode"`
	Echo              EchoSettings        `bson:"echo"`
	NSFWEnabled       bool                `bson:"nsfw_enabled"`
	Links             LinkData            `bson:"links"`
	BioLinkEnabled    bool                `bson:"biolink_enabled"`
	MediaDelete       MediaDeleteSettings `bson:"media_delete"`
	FileDelete        FileDeleteSettings  `bson:"file_delete"`
	MsgDelete         MsgDeleteSettings   `bson:"msg_delete"`
	NoForwardEnabled  bool                `bson:"no_forward_enabled"`
	NoPhoneEnabled    bool                `bson:"no_phone_enabled"`
	NoHashtagsEnabled bool                `bson:"no_hashtags_enabled"`
	NoPromoEnabled    bool                `bson:"no_promo_enabled"`
	NoBotPromoEnabled bool                `bson:"no_bot_promo_enabled"`
	NoStylishEnabled  bool                `bson:"no_stylish_enabled"`
	NoLanguageEnabled bool                `bson:"no_language_enabled"`
	BlockedStickers   []string            `bson:"blocked_stickers"`
	ApprovedUsers     []int64             `bson:"approved_users"`
	NsfwForceSafe     []string            `bson:"nsfw_force_safe"`
	NsfwForceUnsafe   []string            `bson:"nsfw_force_unsafe"`
}

type BotState struct {
	ID            string  `bson:"_id"`
	ServedUsers   []int64 `bson:"served_users"`
	ServedChats   []int64 `bson:"served_chats"`
	LoggerEnabled bool    `bson:"logger_enabled"`
	Migrated      bool    `bson:"migrated"`
}

func (s *BotState) copy() *BotState {
	users := make([]int64, len(s.ServedUsers))
	copy(users, s.ServedUsers)
	chats := make([]int64, len(s.ServedChats))
	copy(chats, s.ServedChats)
	return &BotState{
		ID:            s.ID,
		ServedUsers:   users,
		ServedChats:   chats,
		LoggerEnabled: s.LoggerEnabled,
		Migrated:      s.Migrated,
	}
}
