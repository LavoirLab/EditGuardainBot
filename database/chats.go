package database

import "slices"

func GetServedChats() ([]int64, error) {
	stateMu.RLock()
	defer stateMu.RUnlock()
	state, err := getBotState()
	if err != nil {
		return nil, err
	}
	// Optimization: Only copy the requested slice to avoid unnecessary memory allocations
	// and redundant work from state.copy().
	chats := make([]int64, len(state.ServedChats))
	copy(chats, state.ServedChats)
	return chats, nil
}

func GetServedChatsCount() (int, error) {
	stateMu.RLock()
	defer stateMu.RUnlock()
	state, err := getBotState()
	if err != nil {
		return 0, err
	}
	return len(state.ServedChats), nil
}

func IsServedChat(chatID int64) (bool, error) {
	if IsServedChatCached(chatID) {
		return true, nil
	}
	stateMu.RLock()
	defer stateMu.RUnlock()
	state, err := getBotState()
	if err != nil {
		return false, err
	}
	isServed := slices.Contains(state.ServedChats, chatID)
	if isServed {
		MarkChatServed(chatID)
	}
	return isServed, nil
}

func AddServedChat(chatID int64) error {
	if IsServedChatCached(chatID) {
		return nil
	}

	stateMu.Lock()
	defer stateMu.Unlock()

	state, err := getBotState()
	if err != nil {
		return err
	}

	if !slices.Contains(state.ServedChats, chatID) {
		state.ServedChats = append(state.ServedChats, chatID)
		if err := updateBotState(state); err != nil {
			return err
		}
		MarkChatServed(chatID)
	}
	return nil
}

func DeleteServedChat(chatID int64) error {
	stateMu.Lock()
	defer stateMu.Unlock()

	state, err := getBotState()
	if err != nil {
		return err
	}
	for i, id := range state.ServedChats {
		if id == chatID {
			state.ServedChats = append(
				state.ServedChats[:i],
				state.ServedChats[i+1:]...)
			if err := updateBotState(state); err != nil {
				return err
			}
			seenChats.Delete(chatID)
			return nil
		}
	}
	return nil
}
