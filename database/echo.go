package database

import "strings"

const (
	defaultEchoLimit = 450
	defaultEchoMode  = "ON"
)

func SetEchoLimit(chatID int64, limit int) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}
	if settings.Echo.Limit == limit {
		return nil
	}
	settings.Echo.Limit = limit
	return updateChatSettings(settings)
}

func SetLongMessageStatus(chatID int64, enabled bool) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}
	mode := "OFF"
	if enabled {
		mode = "ON"
	}
	if settings.Echo.Mode == mode {
		return nil
	}
	settings.Echo.Mode = mode
	return updateChatSettings(settings)
}

func GetEchoSettings(chatID int64) (*EchoSettings, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return nil, err
	}

	changed := false
	if settings.Echo.Limit == 0 {
		settings.Echo.Limit = defaultEchoLimit
		changed = true
	}

	mode := strings.ToUpper(settings.Echo.Mode)
	if mode != "ON" && mode != "OFF" {
		settings.Echo.Mode = defaultEchoMode
		changed = true
	}

	if changed {
		updateChatSettings(settings)
	}

	return &settings.Echo, nil
}
