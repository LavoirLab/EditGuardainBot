package database

import (
	"time"
)

func GetMediaDeleteSettings(chatID int64) (*MediaDeleteSettings, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return nil, err
	}
	return &settings.MediaDelete, nil
}

func SetMediaDeleteEnabled(chatID int64, enabled bool) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}
	settings.MediaDelete.Enabled = enabled
	return updateChatSettings(settings)
}

func SetMediaDeleteDelay(chatID int64, delay time.Duration) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}
	settings.MediaDelete.Delay = delay
	return updateChatSettings(settings)
}
