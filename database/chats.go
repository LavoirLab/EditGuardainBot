package database

import "slices"

func GetServedChats() ([]int64, error) {
	state, err := getBotState()
	if err != nil {
		return nil, err
	}
	return state.ServedChats, nil
}

func IsServedChat(chatID int64) (bool, error) {
	chats, err := GetServedChats()
	if err != nil {
		return false, err
	}
	return slices.Contains(chats, chatID), nil
}

func AddServedChat(chatID int64) error {
	state, err := getBotState()
	if err != nil {
		return err
	}
	if !slices.Contains(state.ServedChats, chatID) {
		state.ServedChats = append(state.ServedChats, chatID)
		return updateBotState(state)
	}
	return nil
}

func DeleteServedChat(chatID int64) error {
	state, err := getBotState()
	if err != nil {
		return err
	}
	for i, id := range state.ServedChats {
		if id == chatID {
			state.ServedChats = append(
				state.ServedChats[:i],
				state.ServedChats[i+1:]...)
			return updateBotState(state)
		}
	}
	return nil
}
