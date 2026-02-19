package database

import (
	"strings"
)

const (
	defaultEchoLimit = 450
	defaultEchoMode  = "AUTO"
)

var validModes = map[string]string{
	"AUTO":      "AUTO",
	"AUTOMATIC": "AUTO",
	"MANUAL":    "MANUAL",
	"OFF":       "OFF",
}

func normalizeMode(input string) (string, bool) {
	mode := strings.ToUpper(input)
	val, ok := validModes[mode]
	return val, ok
}

func SetEchoMode(chatID int64, mode string) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}

	if normalized, ok := normalizeMode(mode); ok {
		if settings.Echo.Mode == normalized {
			return nil
		}
		settings.Echo.Mode = normalized
		return updateChatSettings(settings)
	}
	return nil
}

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

func GetEchoSettings(chatID int64) (*EchoSettings, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return nil, err
	}

	changed := false
	// Normalize legacy or incorrect values
	mode := strings.ToUpper(settings.Echo.Mode)
	if normalized, ok := normalizeMode(mode); ok {
		if settings.Echo.Mode != normalized {
			settings.Echo.Mode = normalized
			changed = true
		}
	} else if settings.Echo.Mode != defaultEchoMode {
		settings.Echo.Mode = defaultEchoMode
		changed = true
	}

	if settings.Echo.Limit == 0 {
		settings.Echo.Limit = defaultEchoLimit
		changed = true
	}

	if changed {
		updateChatSettings(settings)
	}

	return &settings.Echo, nil
}
