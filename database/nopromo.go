package database

func IsNoPromoEnabled(chatID int64) (bool, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return false, err
	}
	return settings.NoPromoEnabled, nil
}

func SetNoPromoEnabled(chatID int64, enabled bool) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}
	settings.NoPromoEnabled = enabled
	return updateChatSettings(settings)
}
