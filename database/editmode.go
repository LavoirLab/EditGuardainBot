package database

import "time"

func SetEditMode(chatID int64, enabled bool) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}

	if settings.EditMode.Enabled == enabled {
		return nil
	}

	settings.EditMode.Enabled = enabled
	return updateChatSettings(settings)
}

func SetEditDelay(chatID int64, duration time.Duration) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}

	if settings.EditMode.Duration == duration {
		return nil
	}

	settings.EditMode.Duration = duration
	return updateChatSettings(settings)
}

func GetEditMode(chatID int64) EditModeSettings {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return EditModeSettings{
			Enabled:  true,
			Duration: 0,
		}
	}
	return settings.EditMode
}

func ResetEditMode(chatID int64) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}
	settings.EditMode = EditModeSettings{
		Enabled:  true,
		Duration: 0,
	}
	return updateChatSettings(settings)
}
