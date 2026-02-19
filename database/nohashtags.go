package database

func IsNoHashtagsEnabled(chatID int64) (bool, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return false, err
	}
	return settings.NoHashtagsEnabled, nil
}

func SetNoHashtagsEnabled(chatID int64, enabled bool) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}
	settings.NoHashtagsEnabled = enabled
	return updateChatSettings(settings)
}
