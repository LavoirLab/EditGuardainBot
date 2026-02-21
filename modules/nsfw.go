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
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"
	"main/config"
	"main/database"
	"main/utils"
)

// nsfwFileCache caches NSFW check results by photo file_reference hash (photoID + accessHash)
// key: "<photoID>_<accessHash>" → bool (true = nsfw)
var (
	nsfwFileCache   = utils.NewCache[string, bool](24 * time.Hour)
	nsfwViolations  = utils.NewCache[string, int](24 * time.Hour) // key: "<chatID>_<userID>" → count
	nsfwViolationMu sync.Mutex
)

// nsfwViolationThreshold: how many NSFW images before auto-mute
const nsfwViolationThreshold = 3

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "nsfw",
		ModName: "🔞 NSFW Detection",
		Help: "🔞 <b>NSFW Image Detection</b> automatically detects and removes inappropriate images.\n\n" +
			"<b>Commands:</b>\n" +
			"• <code>/nonsfw</code> — Show module info & current status\n" +
			"• <code>/nonsfw on</code> — Enable NSFW detection ✅\n" +
			"• <code>/nonsfw off</code> — Disable NSFW detection ❌\n\n" +
			"<b>How it works:</b>\n" +
			"• Every photo sent by users is checked against an AI model\n" +
			"• Results are cached per image — same image won't be re-checked\n" +
			"• If NSFW → message is deleted and user is warned\n" +
			"• After <b>3 NSFW violations</b> in 24h → user is <b>muted for 24 hours</b> 🔇\n\n" +
			"<b>ℹ️ Note:</b> Applies to all users (admins are exempt).\n" +
			"👮 Only group admins can enable or disable this feature.\n\n" +
			"<blockquote>⚡ Image results are cached for 24h to avoid duplicate API calls.</blockquote>",
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
	if _, err := m.Delete(); err != nil {
		return err
	}

	args := strings.Fields(m.Args())

	// No args → show status/info
	if len(args) < 1 {
		enabled := database.IsNSFWEnabled(m.ChatID())
		statusEmoji := "❌"
		statusText := "Disabled"
		if enabled {
			statusEmoji = "✅"
			statusText = "Enabled"
		}

		msg := fmt.Sprintf(`🔞 <b>NSFW Detection Module</b>

<b>Status:</b> %s <i>%s</i>

<b>What it does:</b>
• Detects inappropriate images using AI
• Deletes NSFW photos automatically
• Caches results per image (no duplicate API calls)
• Auto-mutes repeat offenders after 3 violations in 24h

<b>Usage:</b>
• <code>/nonsfw on</code> — Enable
• <code>/nonsfw off</code> — Disable`, statusEmoji, statusText)

		_, err := m.Reply(msg)
		return err
	}

	// Has arg → toggle on/off
	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err = m.Reply("❌ Invalid option.\nUse <code>/nonsfw on</code> or <code>/nonsfw off</code>")
		return err
	}

	if err := database.SetNSFWFlag(m.ChatID(), enable); err != nil {
		return err
	}

	statusStr := "disabled ❌"
	if enable {
		statusStr = "enabled ✅"
	}

	_, err = m.Reply(fmt.Sprintf("🔞 <b>NSFW Detection</b> has been <b>%s</b> for this chat.", statusStr))
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

	// Admin exempt
	if isAdmin, _ := utils.IsAdmin(m.Client, m.ChatID(), m.SenderID()); isAdmin {
		return nil
	}

	isNsfw, err := getOrCheckNsfw(m)
	if err != nil {
		m.Client.Log.Error(fmt.Sprintf("NSFW Check Error: %v", err))
		return nil
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

	// Admin exempt
	if len(a.Messages) > 0 {
		if isAdmin, _ := utils.IsAdmin(a.Client, a.ChatID(), a.Messages[0].SenderID()); isAdmin {
			return nil
		}
	}

	isNsfwDetected := false
	for _, m := range a.Messages {
		if m.Photo() == nil {
			continue
		}

		isNsfw, err := getOrCheckNsfw(m)
		if err != nil {
			m.Client.Log.Error(fmt.Sprintf("NSFW Check Error in album: %v", err))
			continue
		}
		if isNsfw {
			isNsfwDetected = true
			break
		}
	}

	if isNsfwDetected {
		_, err := a.Delete()
		if err != nil {
			return err
		}

		sender := a.Messages[0].Sender
		mention := utils.MentionHTML(sender)

		// Track violation and maybe mute
		chatID := a.Messages[0].ChatID()
		userID := a.Messages[0].SenderID()
		muted := trackViolationAndMute(a.Client, chatID, userID)

		warnMsg := buildNsfwWarnMsg(mention, muted)
		_, err = a.Messages[0].Respond(warnMsg, &tg.SendOptions{})
		if err != nil {
			return err
		}
		return tg.ErrEndGroup
	}

	return nil
}

// getOrCheckNsfw checks cache by m.File.FileID, calls API only on cache miss
func getOrCheckNsfw(m *tg.NewMessage) (bool, error) {
	cacheKey := ""
	if m.File != nil {
		cacheKey = m.File.FileID
	}

	// Check cache
	if cacheKey != "" {
		if cached, ok := nsfwFileCache.Get(cacheKey); ok {
			m.Client.Log.Debug(fmt.Sprintf("NSFW cache hit for FileID %s: %v", cacheKey, cached))
			return cached, nil
		}
	}

	// Not cached → call API
	isNsfw, err := checkImageNsfw(m)
	if err != nil {
		return false, err
	}

	// Store in cache
	if cacheKey != "" {
		nsfwFileCache.Set(cacheKey, isNsfw)
	}
	return isNsfw, nil
}

// trackViolationAndMute increments violation count and mutes if threshold reached
// returns true if user was muted
func trackViolationAndMute(client *tg.Client, chatID, userID int64) bool {
	nsfwViolationMu.Lock()
	defer nsfwViolationMu.Unlock()

	key := fmt.Sprintf("%d_%d", chatID, userID)
	count := 0
	if cached, ok := nsfwViolations.Get(key); ok {
		count = cached
	}
	count++
	nsfwViolations.Set(key, count)

	if count >= nsfwViolationThreshold {
		// Mute user for 24 hours
		muteUntil := int32(time.Now().Add(24 * time.Hour).Unix())
		_, err := client.EditBanned(chatID, userID, &tg.BannedOptions{
			Mute:     true,
			TillDate: muteUntil,
		})
		if err != nil {
			client.Log.Error(fmt.Sprintf("NSFW: failed to mute user %d in chat %d: %v", userID, chatID, err))
			return false
		}

		// Reset their count after muting
		nsfwViolations.Set(key, 0)
		return true
	}

	return false
}

func buildNsfwWarnMsg(mention string, muted bool) string {
	base := fmt.Sprintf("🔞 <b>NSFW Content Detected!</b>\n\n<b>User:</b> %s\n<b>Action:</b> <i>Message Deleted</i>", mention)
	if muted {
		base += "\n🔇 <b>Auto-muted for 24 hours</b> due to repeated violations."
	}
	base += "\n\n💡 <i>Please avoid sending NSFW content in this group.</i>"
	return base
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

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
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

	mention := utils.MentionHTML(m.Sender)
	chatID := m.ChatID()
	userID := m.SenderID()

	muted := trackViolationAndMute(m.Client, chatID, userID)
	warnMsg := buildNsfwWarnMsg(mention, muted)

	_, err = m.Respond(warnMsg)
	if err != nil {
		return err
	}
	return tg.ErrEndGroup
}