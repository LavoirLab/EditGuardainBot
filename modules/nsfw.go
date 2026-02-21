package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
	"main/config"
	"main/database"
	"main/utils"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "nsfw",
		ModName: "NSFW Detection",
		Help:    "Control NSFW image detection in the chat.\n\n<b>Commands:</b>\n/nonsfw - Enable/Disable NSFW image detection.",
		Handlers: []HandlerConfig{
			{
				Pattern: "/nonsfw",
				Handler: RequirePermission("change_info", NsfwCmdHandler),
			},
			{
				Pattern: "album",
				Handler: AlbumNsfwHandler,
			},
		},
	})
}

func NsfwCmdHandler(m *tg.NewMessage) error {
	enabled := database.IsNSFWEnabled(m.ChatID())
	newStatus := !enabled
	err := database.SetNSFWFlag(m.ChatID(), newStatus)
	if err != nil {
		return err
	}

	statusStr := "disabled"
	if newStatus {
		statusStr = "enabled"
	}

	_, err = m.Reply(fmt.Sprintf("🛡️ <b>NSFW Detection</b> has been <b>%s</b> for this chat.", statusStr), &tg.SendOptions{})
	return err
}

func NsfwHandler(m *tg.NewMessage) error {
	if !config.NsfwApiEnabled || !database.IsNSFWEnabled(m.ChatID()) {
		return nil
	}

	// Only process single photos here. Albums are handled by AlbumNsfwHandler.
	if m.Photo() == nil || m.Message.GroupedID != 0 {
		return nil
	}

	isNsfw, err := checkImageNsfw(m)
	if err != nil {
		m.Client.Log.Error(fmt.Sprintf("NSFW Check Error: %v", err))
		return nil // Don't block message if API fails
	}

	if isNsfw {
		return handleNsfwDetected(m)
	}

	return nil
}

func AlbumNsfwHandler(a *tg.Album) error {
	if !config.NsfwApiEnabled || !database.IsNSFWEnabled(a.ChatID()) {
		return nil
	}

	isNsfwDetected := false
	for _, m := range a.Messages {
		if m.Photo() != nil {
			isNsfw, err := checkImageNsfw(m)
			if err != nil {
				m.Client.Log.Error(fmt.Sprintf("NSFW Check Error in album: %v", err))
				continue
			}
			if isNsfw {
				isNsfwDetected = true
				break
			}
		}
	}

	if isNsfwDetected {
		// Delete the whole album
		_, err := a.Delete()
		if err != nil {
			return err
		}

		warnMsg := fmt.Sprintf("🛡️ <b>NSFW Content Detected!</b>\n\n<b>User:</b> %s\n<b>Action:</b> <i>Album Deleted</i>\n\n💡 <i>Please avoid sending NSFW content in this group.</i>", utils.MentionHTML(a.Messages[0].Sender))
		_, err = a.Messages[0].Respond(warnMsg, &tg.SendOptions{})
		if err != nil {
			return err
		}
		return tg.ErrEndGroup
	}

	return nil
}

type NsfwApiResponse struct {
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
}

func checkImageNsfw(m *tg.NewMessage) (bool, error) {
	path, err := m.Download()
	if err != nil {
		return false, err
	}
	defer os.Remove(path)

	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return false, err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return false, err
	}
	writer.Close()

	apiUrl := strings.TrimRight(config.NsfwApiUrl, "/") + "/detect"
	req, err := http.NewRequest("POST", apiUrl, body)
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result NsfwApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	return strings.ToLower(result.Label) == "nsfw", nil
}

func handleNsfwDetected(m *tg.NewMessage) error {
	_, err := m.Delete()
	if err != nil {
		return err
	}

	warnMsg := fmt.Sprintf("🛡️ <b>NSFW Content Detected!</b>\n\n<b>User:</b> %s\n<b>Action:</b> <i>Message Deleted</i>\n\n💡 <i>Please avoid sending NSFW content in this group.</i>", utils.MentionHTML(m.Sender))
	_, err = m.Respond(warnMsg, &tg.SendOptions{})
	if err != nil {
		return err
	}
	return tg.ErrEndGroup
}
