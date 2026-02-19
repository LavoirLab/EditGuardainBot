package database

func IsLoggerEnabled() bool {
	state, err := getBotState()
	if err != nil {
		return true // Default
	}
	return state.LoggerEnabled
}

func SetLogger(enabled bool) error {
	state, err := getBotState()
	if err != nil {
		return err
	}
	if state.LoggerEnabled == enabled {
		return nil
	}
	state.LoggerEnabled = enabled
	return updateBotState(state)
}
