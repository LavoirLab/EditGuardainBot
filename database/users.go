package database

import "slices"

func GetServedUsers() ([]int64, error) {
	state, err := getBotState()
	if err != nil {
		return nil, err
	}
	return state.ServedUsers, nil
}

func IsServedUser(userID int64) (bool, error) {
	users, err := GetServedUsers()
	if err != nil {
		return false, err
	}
	return slices.Contains(users, userID), nil
}

func AddServedUser(userID int64) error {
	state, err := getBotState()
	if err != nil {
		return err
	}
	if !slices.Contains(state.ServedUsers, userID) {
		state.ServedUsers = append(state.ServedUsers, userID)
		return updateBotState(state)
	}
	return nil
}

func DeleteServedUser(userID int64) error {
	state, err := getBotState()
	if err != nil {
		return err
	}
	for i, id := range state.ServedUsers {
		if id == userID {
			state.ServedUsers = append(
				state.ServedUsers[:i],
				state.ServedUsers[i+1:]...)
			return updateBotState(state)
		}
	}
	return nil
}
