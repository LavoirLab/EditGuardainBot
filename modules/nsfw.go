package modules

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"

	"main/config"
	"main/database"
	"main/utils"
)

var (
	nsfwFileCache   = utils.NewCache[string, bool](48 * time.Hour)
	nsfwViolations  = utils.NewCache[string, int](24 * time.Hour)
	nsfwViolationMu sync.Mutex
)

const nsfwViolationThreshold = 5

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "nsfw",
		ModName: "🔞 NSFW",
		Help: `<emoji id='5460873805297624194'>🔞</emoji> <b>ɴsғᴡ ᴅᴇᴛᴇᴄᴛɪᴏɴ</b>

<b>ᴄᴏᴍᴍᴀɴᴅs:</b>
<blockquote>• <code>/nonsfw on</code> — ᴇɴᴀʙʟᴇ <emoji id='6084551628461967966'>✅</emoji>
• <code>/nonsfw off</code> — ᴅɪsᴀʙʟᴇ <emoji id='5465665476971471368'>❌</emoji></blockquote>

sᴄᴀɴs ᴘʜᴏᴛᴏs ᴀɴᴅ sᴛɪᴄᴋᴇʀs (ᴡᴇʙᴘ, ᴡᴇʙᴍ). ᴅᴇʟᴇᴛᴇs ɴsғᴡ ᴄᴏɴᴛᴇɴᴛ ᴀɴᴅ ᴍᴜᴛᴇs ʀᴇᴘᴇᴀᴛ ᴏғғᴇɴᴅᴇʀs ᴀғᴛᴇʀ 5 ᴠɪᴏʟᴀᴛɪᴏɴs ɪɴ 1ʜ.`,
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

	if len(args) < 1 {
		enabled := database.IsNSFWEnabled(m.ChatID())
		status := "Disabled ❌"
		if enabled {
			status = "Enabled ✅"
		}
		_, err := m.Reply(
			fmt.Sprintf(
				"🔞 <b>NSFW Detection Status:</b> %s\n\n<i>When enabled, I will scan photos and stickers for adult content.</i>",
				status,
			),
		)
		return err
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err = m.Reply(
			"❗ <b>Invalid usage.</b>\n\nPlease use:\n• <code>/nonsfw on</code> — To enable scanning\n• <code>/nonsfw off</code> — To disable scanning",
		)
		return err
	}

	if err := database.SetNSFWFlag(m.ChatID(), enable); err != nil {
		_, err = m.Reply(
			"⚠️ <b>Oops! Update failed.</b>\n\nSomething went wrong while updating the NSFW settings. Please try again.",
		)
		return err
	}

	status := "disabled ❌"
	next := "Use <code>/nonsfw on</code> to enable it again."
	if enable {
		status = "enabled ✅"
		next = "Use <code>/nonsfw off</code> to disable it."
	}
	_, err = m.Reply(
		fmt.Sprintf("🔞 <b>NSFW Detection %s</b>\n\n%s", status, next),
	)
	return err
}

func NsfwHandler(m *tg.NewMessage) error {
	if !config.NsfwApiEnabled || !database.IsNSFWEnabled(m.ChatID()) {
		return nil
	}

	if m.Message.GroupedID != 0 {
		return nil
	}

	isPhoto := m.Photo() != nil
	isSticker := m.Sticker() != nil

	if !isPhoto && !isSticker {
		return nil
	}

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

	if len(a.Messages) > 0 {
		if isAdmin, _ := utils.IsAdmin(a.Client, a.ChatID(), a.Messages[0].SenderID()); isAdmin {
			return nil
		}
	}

	isNsfwDetected := false
	for _, m := range a.Messages {
		if m.Photo() == nil && m.Sticker() == nil {
			continue
		}
		isNsfw, err := getOrCheckNsfw(m)
		if err != nil {
			m.Client.Log.Error(fmt.Sprintf("NSFW album check error: %v", err))
			continue
		}
		if isNsfw {
			isNsfwDetected = true
			break
		}
	}

	if isNsfwDetected {
		if _, err := a.Delete(); err != nil {
			return err
		}
		chatID := a.Messages[0].ChatID()
		userID := a.Messages[0].SenderID()
		muted := trackViolationAndMute(a.Client, chatID, userID)
		_, err := a.Messages[0].Respond(
			buildNsfwWarnMsg(utils.MentionHTML(a.Messages[0].Sender), muted),
		)
		if err != nil {
			return err
		}
		return tg.ErrEndGroup
	}

	return nil
}

func getOrCheckNsfw(m *tg.NewMessage) (bool, error) {
	cacheKey := ""
	if m.File != nil {
		cacheKey = m.File.FileID
	}

	if cacheKey != "" {
		if cached, ok := nsfwFileCache.Get(cacheKey); ok {
			return cached, nil
		}
	}

	var (
		isNsfw bool
		err    error
	)

	if sticker := m.Sticker(); sticker != nil {
		isNsfw, err = checkStickerNsfw(m, sticker)
	} else {
		isNsfw, err = checkImageNsfw(m)
	}

	if err != nil {
		return false, err
	}

	if cacheKey != "" {
		nsfwFileCache.Set(cacheKey, isNsfw)
	}
	return isNsfw, nil
}

func checkStickerNsfw(m *tg.NewMessage, doc *tg.DocumentObj) (bool, error) {
	switch doc.MimeType {
	case "image/webp":
		return checkWebpSticker(m)
	case "video/webm":
		return checkWebmSticker(m)
	default:
		return false, nil
	}
}

func checkWebpSticker(m *tg.NewMessage) (bool, error) {
	webpPath, err := m.Download()
	if err != nil {
		return false, err
	}
	defer os.Remove(webpPath)

	pngPath := webpPath + ".png"
	defer os.Remove(pngPath)

	if out, err := exec.Command("ffmpeg", "-y", "-i", webpPath, pngPath).CombinedOutput(); err != nil {
		return false, fmt.Errorf("webp→png: %w\n%s", err, out)
	}

	return sendFileToNsfwApi(pngPath)
}

func checkWebmSticker(m *tg.NewMessage) (bool, error) {
	webmPath, err := m.Download()
	if err != nil {
		return false, err
	}
	defer os.Remove(webmPath)

	pngPath := webmPath + "_frame.png"
	defer os.Remove(pngPath)

	if out, err := exec.Command("ffmpeg", "-y", "-i", webmPath, "-vframes", "1", "-f", "image2", pngPath).CombinedOutput(); err != nil {
		return false, fmt.Errorf("webm→frame: %w\n%s", err, out)
	}

	return sendFileToNsfwApi(pngPath)
}

func checkImageNsfw(m *tg.NewMessage) (bool, error) {
	path, err := m.Download()
	if err != nil {
		return false, err
	}
	defer os.Remove(path)

	return sendFileToNsfwApi(path)
}

func sendFileToNsfwApi(filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return false, err
	}
	if _, err = io.Copy(part, file); err != nil {
		return false, err
	}
	writer.Close()

	apiUrl := strings.TrimRight(config.NsfwApiUrl, "/") + "/detect"
	req, err := http.NewRequest("POST", apiUrl, body)
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("API status %d", resp.StatusCode)
	}

	var result NsfwApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	fmt.Println("The confidence level is ", result.Confidence)
	return strings.ToLower(result.Label) == "nsfw", nil
}

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
		muteUntil := int32(time.Now().Add(1 * time.Hour).Unix())
		_, err := client.EditBanned(chatID, userID, &tg.BannedOptions{
			Mute:     true,
			TillDate: muteUntil,
		})
		if err != nil {
			client.Log.Error(
				fmt.Sprintf(
					"NSFW mute failed for user %d in chat %d: %v",
					userID,
					chatID,
					err,
				),
			)
			return false
		}
		nsfwViolations.Set(key, 0)
		return true
	}

	return false
}

func buildNsfwWarnMsg(mention string, muted bool) string {
	msg := fmt.Sprintf(
		"🔞 <b>NSFW Content Blocked!</b>\n\n%s, <i>your message was removed because it contained suggestive or adult content.</i>",
		mention,
	)
	if muted {
		msg += "\n\n🔇 <b>Note:</b> You have been muted for 1 hour due to repeated violations of the safety policy."
	}
	return msg
}

type NsfwApiResponse struct {
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
}

func handleNsfwDetected(m *tg.NewMessage) error {
	if _, err := m.Delete(); err != nil {
		return err
	}

	muted := trackViolationAndMute(m.Client, m.ChatID(), m.SenderID())
	_, err := m.Respond(buildNsfwWarnMsg(utils.MentionHTML(m.Sender), muted))
	if err != nil {
		return err
	}
	return tg.ErrEndGroup
}
