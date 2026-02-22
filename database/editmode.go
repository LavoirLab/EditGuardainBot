package database

func SetEditMode(chatID int64, mode string, duration int) (bool, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return false, err
	}

	if settings.EditMode.Mode == mode &&
		settings.EditMode.Duration == duration {
		return true, nil
	}

	settings.EditMode.Mode = mode
	settings.EditMode.Duration = duration

	err = updateChatSettings(settings)
	return err == nil, err
}

func GetEditMode(chatID int64) EditModeSettings {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return EditModeSettings{
			Mode:     "USER",
			Duration: 0,
		}
	}
	return settings.EditMode
}

func ResetEditMode(chatID int64) error {
	_, err := SetEditMode(chatID, "USER", 0)
	return err
}
