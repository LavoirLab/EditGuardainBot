package database

func IsNoPhoneEnabled(chatID int64) (bool, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return false, err
	}
	return settings.NoPhoneEnabled, nil
}

func SetNoPhoneEnabled(chatID int64, enabled bool) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}
	settings.NoPhoneEnabled = enabled
	return updateChatSettings(settings)
}
