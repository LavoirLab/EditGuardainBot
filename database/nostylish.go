package database

func IsNoStylishEnabled(chatID int64) (bool, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return false, err
	}
	return settings.NoStylishEnabled, nil
}

func SetNoStylishEnabled(chatID int64, enabled bool) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}
	settings.NoStylishEnabled = enabled
	return updateChatSettings(settings)
}
