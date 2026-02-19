package database

func IsNoForwardEnabled(chatID int64) (bool, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return false, err
	}
	return settings.NoForwardEnabled, nil
}

func SetNoForwardEnabled(chatID int64, enabled bool) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}
	settings.NoForwardEnabled = enabled
	return updateChatSettings(settings)
}
