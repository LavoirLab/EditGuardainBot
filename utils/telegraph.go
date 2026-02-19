package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

const TelegraphApi = "https://api.telegra.ph/"

var AccountMap map[string]int64

type TelegraphAccountResponse struct {
	OK     bool `json:"ok"`
	Result struct {
		AccessToken string `json:"access_token"`
	} `json:"result"`
}

type TelegraphResponse struct {
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	Result struct {
		URL string `json:"url"`
	} `json:"result"`
}

func createAccount(shortName string) (string, error) {
	payload := map[string]string{
		"short_name":  shortName,
		"author_name": "Anonymous",
		"author_url":  "https://t.me/ViyomBot",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", TelegraphApi+"createAccount", bytes.NewReader(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result TelegraphAccountResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", err
	}

	if !result.OK {
		return "", errors.New("failed to create account")
	}

	return result.Result.AccessToken, nil
}

func init() {
	AccountMap = make(map[string]int64)
	for i := 0; i < 3; i++ {
		token, err := createAccount("EcoBot" + strconv.Itoa(i+1))
		if err == nil {
			AccountMap[token] = 0
		}
	}
}

func getAvailableToken() (string, error) {
	now := time.Now().Unix()

	for token, waitTime := range AccountMap {
		if waitTime == 0 || waitTime <= now {
			return token, nil
		}
	}

	newToken, err := createAccount("EcoBot" + strconv.Itoa(len(AccountMap)+1))
	if err != nil {
		return "", errors.New("no available accounts and failed to create new account")
	}

	AccountMap[newToken] = 0
	return newToken, nil
}

func extractFloodWait(errorMsg string) int64 {
	re := regexp.MustCompile(`FLOOD_WAIT_(\d+)`)
	matches := re.FindStringSubmatch(errorMsg)
	if len(matches) > 1 {
		seconds, err := strconv.ParseInt(matches[1], 10, 64)
		if err == nil {
			return seconds
		}
	}
	return 0
}

func CreateTelegraphPage(content, firstName, authorURL string) (string, error) {
	for {
		accessToken, err := getAvailableToken()
		if err != nil {
			log.Printf("Telegraph Token error: %v", err)
			return "", err
		}

		telegraphContent := []map[string]interface{}{
			{"tag": "p", "children": []string{content}},
		}

		payload := map[string]interface{}{
			"access_token": accessToken,
			"title":        "Eco Message",
			"author_name":  firstName,
			"author_url":   authorURL,
			"content":      telegraphContent,
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			log.Printf("Telegraph json marshal error: %v", err)
			return "", err

		}

		req, err := http.NewRequest("POST", TelegraphApi+"createPage", bytes.NewReader(jsonData))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{Timeout: 10 * time.Second}

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Telegraph api error: %v", err)

			return "", err
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		var result TelegraphResponse
		if err := json.Unmarshal(bodyBytes, &result); err != nil {
			return "", err
		}

		if result.OK {
			return result.Result.URL, nil
		}

		floodWaitTime := extractFloodWait(result.Error)
		if floodWaitTime > 0 {
			AccountMap[accessToken] = time.Now().Unix() + floodWaitTime
			continue
		}

		log.Printf("Telegraph Error error: %s", result.Error)

		return "", errors.New(result.Error)
	}
}
