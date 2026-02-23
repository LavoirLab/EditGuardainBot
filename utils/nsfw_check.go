package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var nsfwHealthClient = &http.Client{
	Timeout: 5 * time.Second,
}

type NsfwHealthResponse struct {
	Status string `json:"status"`
}

func CheckNsfwApi(url string) error {
	if url == "" {
		return fmt.Errorf("NSFW API URL is empty")
	}

	resp, err := nsfwHealthClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("NSFW API returned status: %d", resp.StatusCode)
	}

	var health NsfwHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return fmt.Errorf("failed to decode NSFW API response: %v", err)
	}

	if health.Status != "online" && health.Status != "healthy" {
		return fmt.Errorf("NSFW API status is neither 'online' nor 'healthy': %s", health.Status)
	}

	return nil
}
