package database

func IsNoLanguageEnabled(chatID int64) (bool, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return false, err
	}
	return settings.NoLanguageEnabled, nil
}

func SetNoLanguageEnabled(chatID int64, enabled bool) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}
	settings.NoLanguageEnabled = enabled
	return updateChatSettings(settings)
}
