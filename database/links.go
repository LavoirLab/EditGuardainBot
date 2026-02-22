package database

import "slices"

func GetAllowedHostnames(chatID int64) ([]string, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return nil, err
	}
	return settings.Links.Hostnames, nil
}

func AddAllowedHostname(chatID int64, hostname string) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}
	if !slices.Contains(settings.Links.Hostnames, hostname) {
		settings.Links.Hostnames = append(settings.Links.Hostnames, hostname)
		return updateChatSettings(settings)
	}
	return nil
}

func RemoveAllowedHostname(chatID int64, hostname string) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}
	for i, h := range settings.Links.Hostnames {
		if h == hostname {
			settings.Links.Hostnames = append(
				settings.Links.Hostnames[:i],
				settings.Links.Hostnames[i+1:]...)
			return updateChatSettings(settings)
		}
	}
	return nil
}

func IsLinkFilterEnabled(chatID int64) (bool, error) {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return false, err
	}
	return settings.Links.Enabled, nil
}

func SetLinkFilterEnabled(chatID int64, enabled bool) error {
	settings, err := getChatSettings(chatID)
	if err != nil {
		return err
	}
	settings.Links.Enabled = enabled
	return updateChatSettings(settings)
}
