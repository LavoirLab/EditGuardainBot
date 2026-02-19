package database

import (
	"time"
)

func GetMsgDeleteSettings(chatID int64) (*MsgDeleteSettings, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return nil, err
	}
	return &settings.MsgDelete, nil
}

func SetMsgDeleteEnabled(chatID int64, enabled bool) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}
	settings.MsgDelete.Enabled = enabled
	return updateChatSettings(settings)
}

func SetMsgDeleteDelay(chatID int64, delay time.Duration) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}
	settings.MsgDelete.Delay = delay
	return updateChatSettings(settings)
}
