package database

func SetNSFWFlag(chatID int64, enable bool) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}

	if settings.NSFWEnabled == enable {
		return nil
	}

	settings.NSFWEnabled = enable
	return updateChatSettings(settings)
}

func IsNSFWEnabled(chatID int64) bool {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return false
	}
	return settings.NSFWEnabled
}
