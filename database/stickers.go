package database

import "slices"

func BlockSticker(chatID int64, fileID string) (bool, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return false, err
	}
	if slices.Contains(settings.BlockedStickers, fileID) {
		return false, nil
	}
	settings.BlockedStickers = append(settings.BlockedStickers, fileID)
	err = updateChatSettings(settings)
	return err == nil, err
}

func UnblockSticker(chatID int64, fileID string) (bool, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return false, err
	}
	for i, id := range settings.BlockedStickers {
		if id == fileID {
			settings.BlockedStickers = append(
				settings.BlockedStickers[:i],
				settings.BlockedStickers[i+1:]...,
			)
			err = updateChatSettings(settings)
			return err == nil, err
		}
	}
	return false, nil
}

func IsStickerBlocked(chatID int64, fileID string) (bool, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return false, err
	}
	return slices.Contains(settings.BlockedStickers, fileID), nil
}

func GetBlockedStickers(chatID int64) ([]string, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return nil, err
	}
	return settings.BlockedStickers, nil
}
