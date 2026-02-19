package database

func SetBioMode(chatID int64) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}
	settings.BioLinkEnabled = true
	return updateChatSettings(settings)
}

func GetBioMode(chatID int64) (bool, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return false, err
	}
	return settings.BioLinkEnabled, nil
}

func DelBioMode(chatID int64) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}
	settings.BioLinkEnabled = false
	return updateChatSettings(settings)
}
