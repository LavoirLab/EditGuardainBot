package database

import (
	"time"
)

func GetFileDeleteSettings(chatID int64) (*FileDeleteSettings, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return nil, err
	}
	return &settings.FileDelete, nil
}

func SetFileDeleteEnabled(chatID int64, enabled bool) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}
	settings.FileDelete.Enabled = enabled
	return updateChatSettings(settings)
}

func SetFileDeleteDelay(chatID int64, delay time.Duration) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}
	settings.FileDelete.Delay = delay
	return updateChatSettings(settings)
}
