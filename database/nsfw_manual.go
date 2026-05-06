package database

import "slices"

func MarkNsfwForceSafe(chatID int64, fileID string) (bool, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return false, err
	}
	if slices.Contains(settings.NsfwForceSafe, fileID) {
		return false, nil
	}
	settings.NsfwForceSafe = append(settings.NsfwForceSafe, fileID)
	for i, id := range settings.NsfwForceUnsafe {
		if id == fileID {
			settings.NsfwForceUnsafe = append(
				settings.NsfwForceUnsafe[:i],
				settings.NsfwForceUnsafe[i+1:]...,
			)
			break
		}
	}
	err = updateChatSettings(settings)
	return err == nil, err
}

func MarkNsfwForceUnsafe(chatID int64, fileID string) (bool, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return false, err
	}
	if slices.Contains(settings.NsfwForceUnsafe, fileID) {
		return false, nil
	}
	settings.NsfwForceUnsafe = append(settings.NsfwForceUnsafe, fileID)
	for i, id := range settings.NsfwForceSafe {
		if id == fileID {
			settings.NsfwForceSafe = append(
				settings.NsfwForceSafe[:i],
				settings.NsfwForceSafe[i+1:]...,
			)
			break
		}
	}
	err = updateChatSettings(settings)
	return err == nil, err
}

func UnsetNsfwManual(chatID int64, fileID string) (bool, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return false, err
	}
	changed := false
	for i, id := range settings.NsfwForceSafe {
		if id == fileID {
			settings.NsfwForceSafe = append(
				settings.NsfwForceSafe[:i],
				settings.NsfwForceSafe[i+1:]...,
			)
			changed = true
			break
		}
	}
	for i, id := range settings.NsfwForceUnsafe {
		if id == fileID {
			settings.NsfwForceUnsafe = append(
				settings.NsfwForceUnsafe[:i],
				settings.NsfwForceUnsafe[i+1:]...,
			)
			changed = true
			break
		}
	}
	if !changed {
		return false, nil
	}
	err = updateChatSettings(settings)
	return err == nil, err
}

func GetNsfwManualLists(chatID int64) ([]string, []string, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return nil, nil, err
	}
	return settings.NsfwForceSafe, settings.NsfwForceUnsafe, nil
}

func GetNsfwManualDecision(chatID int64, fileID string) (int, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return 0, err
	}
	if slices.Contains(settings.NsfwForceSafe, fileID) {
		return -1, nil
	}
	if slices.Contains(settings.NsfwForceUnsafe, fileID) {
		return 1, nil
	}
	return 0, nil
}
