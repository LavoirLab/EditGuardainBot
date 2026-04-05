package database

func IsNoBotPromoEnabled(chatID int64) (bool, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return false, err
	}
	return settings.NoBotPromoEnabled, nil
}

func SetNoBotPromoEnabled(chatID int64, enabled bool) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}
	settings.NoBotPromoEnabled = enabled
	return updateChatSettings(settings)
}
