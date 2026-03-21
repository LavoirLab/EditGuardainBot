package database

import "slices"

func GetServedUsers() ([]int64, error) {
	stateMu.RLock()
	defer stateMu.RUnlock()
	state, err := getBotState()
	if err != nil {
		return nil, err
	}
	// Optimization: Only copy the requested slice to avoid unnecessary memory allocations
	// and redundant work from state.copy().
	users := make([]int64, len(state.ServedUsers))
	copy(users, state.ServedUsers)
	return users, nil
}

func GetServedUsersCount() (int, error) {
	stateMu.RLock()
	defer stateMu.RUnlock()
	state, err := getBotState()
	if err != nil {
		return 0, err
	}
	return len(state.ServedUsers), nil
}

func IsServedUser(userID int64) (bool, error) {
	if IsServedUserCached(userID) {
		return true, nil
	}
	stateMu.RLock()
	defer stateMu.RUnlock()
	state, err := getBotState()
	if err != nil {
		return false, err
	}
	isServed := slices.Contains(state.ServedUsers, userID)
	if isServed {
		MarkUserServed(userID)
	}
	return isServed, nil
}

func AddServedUser(userID int64) error {
	if IsServedUserCached(userID) {
		return nil
	}

	stateMu.Lock()
	defer stateMu.Unlock()

	state, err := getBotState()
	if err != nil {
		return err
	}

	if !slices.Contains(state.ServedUsers, userID) {
		state.ServedUsers = append(state.ServedUsers, userID)
		if err := updateBotState(state); err != nil {
			return err
		}
		MarkUserServed(userID)
	}
	return nil
}

func DeleteServedUser(userID int64) error {
	stateMu.Lock()
	defer stateMu.Unlock()

	state, err := getBotState()
	if err != nil {
		return err
	}
	for i, id := range state.ServedUsers {
		if id == userID {
			state.ServedUsers = append(
				state.ServedUsers[:i],
				state.ServedUsers[i+1:]...)
			if err := updateBotState(state); err != nil {
				return err
			}
			seenUsers.Delete(userID)
			return nil
		}
	}
	return nil
}
