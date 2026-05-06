package database

import "slices"

func IsUserApproved(chatID, userID int64) (bool, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return false, err
	}
	return slices.Contains(settings.ApprovedUsers, userID), nil
}

func ApproveUser(chatID, userID int64) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}

	if !slices.Contains(settings.ApprovedUsers, userID) {
		settings.ApprovedUsers = append(settings.ApprovedUsers, userID)
	}

	return updateChatSettings(settings)
}

func UnapproveUser(chatID, userID int64) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}

	for i, id := range settings.ApprovedUsers {
		if id == userID {
			settings.ApprovedUsers = append(
				settings.ApprovedUsers[:i],
				settings.ApprovedUsers[i+1:]...)
			break
		}
	}

	return updateChatSettings(settings)
}
