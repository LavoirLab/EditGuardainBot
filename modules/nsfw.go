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
	"strconv"
	"strings"
	"sync"
	"time"

	tg "github.com/amarnathcjd/gogram/telegram"

	"main/config"
	"main/database"
	"main/utils"
)

var httpClient = &http.Client{
	Timeout: 90 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        20,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	},
}

const (
	nsfwViolationThreshold = 5
	nsfwMaxConcurrent      = 8
)

var (
	nsfwFileCache    = utils.NewCache[string, bool](48 * time.Hour)
	nsfwViolations   = utils.NewCache[nsfwViolationKey, int](24 * time.Hour)
	nsfwViolationsMu sync.Mutex
	nsfwSemaphore    = make(chan struct{}, nsfwMaxConcurrent)
)

type nsfwViolationKey struct {
	ChatID int64
	UserID int64
}

type nsfwMediaType string

const (
	mediaTypePhoto         nsfwMediaType = "photo"
	mediaTypeStickerStatic nsfwMediaType = "sticker_static"
	mediaTypeStickerVideo  nsfwMediaType = "sticker_video"
	mediaTypeVideo         nsfwMediaType = "video"
)

const (
	nsfwHFScanAPIURL      = "https://nexacoders-nexa-api.hf.space/scan"
	nsfwAPI4AIURL         = "https://demo.api4ai.cloud/nsfw/v1/results"
	nsfwAPI4AIKey         = ""
	nsfwFallbackThreshold = 60.0
)

func getNsfwDashboard(enabled bool) (string, tg.ReplyMarkup) {
	status := "Disabled ❌"
	toggleLabel := "Enable ✅"
	toggleVal := "on"
	if enabled {
		status = "Enabled ✅"
		toggleLabel = "Disable ❌"
		toggleVal = "off"
	}

	text := fmt.Sprintf("🔞 <b>NSFW Detection</b>\n"+
		"━━━━━━━━━━━━━━━━━━━━\n"+
		"<b>Status:</b> %s\n\n"+
		"<blockquote expandable>"+
		"When enabled, I scan photos, videos, and stickers (WebP, WebM) for adult content.\n\n"+
		"<b>Action:</b> NSFW content is deleted immediately, and repeat offenders are muted for 1 hour after 5 violations."+
		"</blockquote>\n\n"+
		"💡 <b>Tip:</b> Keeping this enabled helps maintain a safe environment for all members.", status)

	keyboard := tg.NewKeyboard().AddRow(
		tg.Button.Data(toggleLabel, "nsfw_toggle:"+toggleVal),
		tg.Button.Data("🗑️ Close", "close"),
	).Build()

	return text, keyboard
}

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "nsfw",
		ModName: "🔞 NSFW",
		Help: `<emoji id='5460873805297624194'>🔞</emoji> <b>ɴsғᴡ ᴅᴇᴛᴇᴄᴛɪᴏɴ</b>

<i>sᴄᴀɴs ᴘʜᴏᴛᴏs, ᴠɪᴅᴇᴏs ᴀɴᴅ sᴛɪᴄᴋᴇʀs (ᴡᴇʙᴘ, ᴡᴇʙᴍ). ᴅᴇʟᴇᴛᴇs ɴsғᴡ ᴄᴏɴᴛᴇɴᴛ ᴀɴᴅ ᴍᴜᴛᴇs ʀᴇᴘᴇᴀᴛ ᴏғғᴇɴᴅᴇʀs ᴀғᴛᴇʀ 5 ᴠɪᴏʟᴀᴛɪᴏɴs ɪɴ 1ʜ.</i>

<b>ᴄᴏᴍᴍᴀɴᴅs:</b>
<blockquote>• <code>/nonsfw on</code> — ᴇɴᴀʙʟᴇ <emoji id='6084551628461967966'>✅</emoji>
• <code>/nonsfw off</code> — ᴅɪsᴀʙʟᴇ <emoji id='5465665476971471368'>❌</emoji></blockquote>`,
		Handlers: []HandlerConfig{
			{
				Pattern: "command:nonsfw",
				Handler: RequirePermission("delete_messages", NsfwCmdHandler),
			},
			{
				Pattern: "callback:^nsfw_toggle:(on|off)$",
				Handler: handleNsfwToggle,
			},
			{
				Pattern: "album",
				Handler: AlbumNsfwHandler,
			},
			{
				Pattern: "command:(nsfwsafe|nsfwsfw)",
				Handler: RequirePermission("delete_messages", NsfwMarkSafeCmd),
			},
			{
				Pattern: "command:(nsfwunsafe|nsfwmark)",
				Handler: RequirePermission(
					"delete_messages",
					NsfwMarkUnsafeCmd,
				),
			},
			{
				Pattern: "command:(nsfwreset|nsfwrevert)",
				Handler: RequirePermission("delete_messages", NsfwResetCmd),
			},
			{
				Pattern: "command:(nsfwlist|nsfwmanual)",
				Handler: NsfwManualListCmd,
			},
		},
	})
}

func handleNsfwToggle(c *tg.CallbackQuery) error {
	if isAdmin, _ := utils.IsAdmin(c.Client, c.ChatID, c.SenderID); !isAdmin {
		c.Answer("👤 Admin Access Required", &tg.CallbackOptions{Alert: true})
		return tg.ErrEndGroup
	}

	parts := strings.Split(c.DataString(), ":")
	enable := parts[1] == "on"

	if err := database.SetNSFWFlag(c.ChatID, enable); err != nil {
		c.Answer("⚠️ Update failed.", &tg.CallbackOptions{Alert: true})
		return err
	}

	text, keyboard := getNsfwDashboard(enable)
	c.Answer("✓ Settings Updated")
	_, err := c.Edit(text, &tg.SendOptions{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	})
	return err
}

func NsfwCmdHandler(m *tg.NewMessage) error {
	if _, err := m.Delete(); err != nil {
		return err
	}

	args := strings.Fields(m.Args())

	if len(args) < 1 {
		enabled := database.IsNSFWEnabled(m.ChatID())
		text, keyboard := getNsfwDashboard(enabled)
		_, err := m.Reply(text, &tg.SendOptions{
			ReplyMarkup: keyboard,
			ParseMode:   "HTML",
		})
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

	text, keyboard := getNsfwDashboard(enable)
	_, err = m.Reply(text, &tg.SendOptions{
		ReplyMarkup: keyboard,
		ParseMode:   "HTML",
	})
	return err
}

func NsfwHandler(m *tg.NewMessage) error {
	if !database.IsNSFWEnabled(m.ChatID()) {
		return nil
	}

	if m.Message.GroupedID != 0 {
		return nil
	}

	mt := resolveMediaType(m)
	if mt == "" {
		return nil
	}

	if isAdmin, _ := utils.IsAdmin(m.Client, m.ChatID(), m.SenderID()); isAdmin {
		return nil
	}

	go func() {
		nsfwSemaphore <- struct{}{}
		defer func() { <-nsfwSemaphore }()

		isNsfw, err := getOrCheckNsfw(m, mt)
		if err != nil {
			m.Client.Log.Error(fmt.Sprintf("NSFW check error: %v", err))
			return
		}
		if isNsfw {
			if err := handleNsfwDetected(m); err != nil &&
				err != tg.ErrEndGroup {
				m.Client.Log.Error(
					fmt.Sprintf("NSFW detection handler error: %v", err),
				)
			}
		}
	}()

	return nil
}

type pendingMedia struct {
	msg       *tg.NewMessage
	path      string
	mediaType nsfwMediaType
}

func shouldScanAlbum(a *tg.Album) bool {
	if !database.IsNSFWEnabled(a.ChatID()) {
		return false
	}

	if len(a.Messages) > 0 {
		if isAdmin, _ := utils.IsAdmin(a.Client, a.ChatID(), a.Messages[0].SenderID()); isAdmin {
			return false
		}
	}

	return true
}

func checkVideoMediaNsfw(m *tg.NewMessage, mt nsfwMediaType) (bool, error) {
	var isNsfw bool
	var err error

	if mt == mediaTypeStickerVideo {
		isNsfw, err = checkStickerVideoNsfw(m)
	} else if mt == mediaTypeVideo {
		isNsfw, err = extractVideoFramesNsfw(m)
	} else {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	if m.File != nil {
		nsfwFileCache.Set(m.File.FileID, isNsfw)
	}
	return isNsfw, nil
}

func processAlbumMedia(a *tg.Album, allPaths *[]string) (bool, []pendingMedia) {
	var pending []pendingMedia
	isNsfwDetected := false

	for _, m := range a.Messages {
		mt := resolveMediaType(m)
		if mt == "" {
			continue
		}

		if m.File != nil {
			if cached, ok := nsfwFileCache.Get(m.File.FileID); ok {
				if cached {
					isNsfwDetected = true
					break
				}
				continue
			}
		}

		if mt == mediaTypeStickerVideo || mt == mediaTypeVideo {
			isNsfw, err := checkVideoMediaNsfw(m, mt)
			if err != nil {
				a.Client.Log.Error(
					fmt.Sprintf("NSFW video error for msg %d: %v", m.ID, err),
				)
				continue
			}
			if isNsfw {
				isNsfwDetected = true
				break
			}
			continue
		}

		path, err := m.Download()
		if err != nil {
			a.Client.Log.Error(
				fmt.Sprintf(
					"NSFW album download error for msg %d: %v",
					m.ID,
					err,
				),
			)
			continue
		}
		*allPaths = append(*allPaths, path)
		pending = append(
			pending,
			pendingMedia{msg: m, path: path, mediaType: mt},
		)
	}

	return isNsfwDetected, pending
}

func AlbumNsfwHandler(a *tg.Album) error {
	if !shouldScanAlbum(a) {
		return nil
	}

	go func() {
		nsfwSemaphore <- struct{}{}
		defer func() { <-nsfwSemaphore }()

		var allPaths []string
		defer func() {
			for _, p := range allPaths {
				os.Remove(p)
			}
		}()

		isNsfwDetected, pending := processAlbumMedia(a, &allPaths)

		if !isNsfwDetected && len(pending) > 0 {
			isNsfwDetected = performBatchScan(a, pending)
		}

		if isNsfwDetected {
			handleDetectedNsfwAlbum(a)
		}
	}()

	return nil
}

func performBatchScan(a *tg.Album, pending []pendingMedia) bool {
	var items []batchEntry
	for _, p := range pending {
		items = append(
			items,
			batchEntry{path: p.path, mediaType: p.mediaType},
		)
	}

	var batchResults []NsfwApiResponse
	var err error
	if config.NsfwOnLocalhost {
		batchResults, err = sendPathBatchToNsfwApi(items)
	} else {
		batchResults, err = sendBatchToNsfwApi(items)
	}

	isNsfwDetected := false
	if err != nil {
		a.Client.Log.Error(fmt.Sprintf("NSFW album batch API error: %v", err))
	} else {
		for i, res := range batchResults {
			isNsfw := strings.ToLower(res.Label) == "nsfw"
			if i < len(pending) && pending[i].msg.File != nil {
				nsfwFileCache.Set(pending[i].msg.File.FileID, isNsfw)
			}
			if isNsfw {
				isNsfwDetected = true
			}
		}
	}
	return isNsfwDetected
}

func handleDetectedNsfwAlbum(a *tg.Album) {
	if _, err := a.Delete(); err != nil {
		return
	}
	chatID := a.Messages[0].ChatID()
	userID := a.Messages[0].SenderID()
	muted := trackViolationAndMute(a.Client, chatID, userID)
	mm, _ := a.Messages[0].Respond(
		buildNsfwWarnMsg(
			utils.MentionHTML(a.Messages[0].Sender),
			muted,
		),
	)
	if mm != nil {
		ScheduleDeletion(mm, 30*time.Second)
	}
}

func resolveMediaType(m *tg.NewMessage) nsfwMediaType {
	if m.Photo() != nil {
		return mediaTypePhoto
	}

	if sticker := m.Sticker(); sticker != nil {
		switch sticker.MimeType {
		case "application/x-tgsticker":
			return ""
		case "video/webm":
			return mediaTypeStickerVideo
		default:
			return mediaTypeStickerStatic
		}
	}

	if m.Animation() != nil {
		return mediaTypeVideo
	}

	if m.Video() != nil {
		return mediaTypeVideo
	}

	return ""
}

func getOrCheckNsfw(m *tg.NewMessage, mt nsfwMediaType) (bool, error) {
	cacheKey := ""
	if m.File != nil {
		cacheKey = m.File.FileID
	}

	if cacheKey != "" {
		decision, err := database.GetNsfwManualDecision(m.ChatID(), cacheKey)
		if err == nil {
			if decision < 0 {
				return false, nil
			}
			if decision > 0 {
				return true, nil
			}
		}
		if cached, ok := nsfwFileCache.Get(cacheKey); ok {
			return cached, nil
		}
	}

	var isNsfw bool
	var err error

	if config.NsfwOnLocalhost {
		isNsfw, err = checkNsfwLocalhost(m, mt)
	} else {
		isNsfw, err = checkNsfwRemote(m, mt)
	}

	if err != nil {
		return false, err
	}

	if cacheKey != "" {
		nsfwFileCache.Set(cacheKey, isNsfw)
	}
	return isNsfw, nil
}

func checkNsfwLocalhost(m *tg.NewMessage, mt nsfwMediaType) (bool, error) {
	if mt == mediaTypeStickerVideo {
		return checkStickerVideoNsfw(m)
	}

	path, err := m.Download()
	if err != nil {
		return false, err
	}
	defer os.Remove(path)

	return sendPathToNsfwApi(path, mt)
}

func checkNsfwRemote(m *tg.NewMessage, mt nsfwMediaType) (bool, error) {
	switch {
	case mt == mediaTypeStickerVideo:
		return checkStickerVideoNsfw(m)
	case mt == mediaTypeVideo:
		return extractVideoFramesNsfw(m)
	default:
		path, err := m.Download()
		if err != nil {
			return false, err
		}
		defer os.Remove(path)
		return sendFileToNsfwApi(path, mt)
	}
}

func checkStickerVideoNsfw(m *tg.NewMessage) (bool, error) {
	webmPath, err := m.Download()
	if err != nil {
		return false, err
	}
	defer os.Remove(webmPath)

	jpgPath := webmPath + "_frame.jpg"
	defer os.Remove(jpgPath)

	cmd := exec.Command(
		"ffmpeg", "-y",
		"-i", webmPath,
		"-vf", "select=eq(n\\,0)",
		"-vsync", "vfr",
		"-frames:v", "1",
		"-q:v", "2",
		jpgPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf(
			"ffmpeg sticker frame extraction failed: %w\n%s",
			err,
			out,
		)
	}

	return sendFileToNsfwApi(jpgPath, mediaTypePhoto)
}

func extractVideoFramesNsfw(m *tg.NewMessage) (bool, error) {
	videoPath, err := m.Download()
	if err != nil {
		return false, err
	}
	defer os.Remove(videoPath)

	tmpDir, err := os.MkdirTemp("", "nsfw_frames_*")
	if err != nil {
		return false, err
	}
	defer os.RemoveAll(tmpDir)

	framePattern := filepath.Join(tmpDir, "frame_%02d.jpg")
	cmd := exec.Command(
		"ffmpeg",
		"-y",
		"-i",
		videoPath,
		"-vf",
		"select='not(mod(n\\,trunc(nb_frames/6+1)))',setpts=N/FRAME_RATE/TB",
		"-vsync",
		"vfr",
		"-frames:v",
		"6",
		"-q:v",
		"2",
		framePattern,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf(
			"ffmpeg frame extraction failed: %w\n%s",
			err,
			out,
		)
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil || len(entries) == 0 {
		return false, fmt.Errorf("no frames extracted from video")
	}

	var items []batchEntry
	for _, e := range entries {
		items = append(items, batchEntry{
			path:      filepath.Join(tmpDir, e.Name()),
			mediaType: mediaTypePhoto,
		})
	}

	results, err := sendBatchToNsfwApi(items)
	if err != nil {
		return false, err
	}

	for _, r := range results {
		if strings.ToLower(r.Label) == "nsfw" {
			return true, nil
		}
	}
	return false, nil
}

func sendFileToNsfwApi(filePath string, mt nsfwMediaType) (bool, error) {
	if config.NsfwApiEnabled && config.NsfwApiUrl != "" {
		isNsfw, err := sendFileToPrimaryNsfwApi(filePath, mt)
		if err == nil {
			return isNsfw, nil
		}
	}
	return sendFileToFallbackApis(filePath)
}

func sendFileToPrimaryNsfwApi(filePath string, mt nsfwMediaType) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, fmt.Errorf("open %s: %w", filePath, err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("media_type", string(mt)); err != nil {
		return false, err
	}

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return false, err
	}
	if _, err = io.Copy(part, file); err != nil {
		return false, err
	}
	writer.Close()

	apiURL := strings.TrimRight(config.NsfwApiUrl, "/") + "/detect"
	req, err := http.NewRequest(http.MethodPost, apiURL, body)
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf(
			"API status %d: %s",
			resp.StatusCode,
			string(bodyBytes),
		)
	}

	var result NsfwApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	return strings.ToLower(result.Label) == "nsfw", nil
}

func sendPathToNsfwApi(filePath string, mt nsfwMediaType) (bool, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("media_type", string(mt)); err != nil {
		return false, err
	}
	if err := writer.WriteField("file_path", filePath); err != nil {
		return false, err
	}
	writer.Close()

	apiURL := strings.TrimRight(config.NsfwApiUrl, "/") + "/detect"
	req, err := http.NewRequest(http.MethodPost, apiURL, body)
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf(
			"API status %d: %s",
			resp.StatusCode,
			string(bodyBytes),
		)
	}

	var result NsfwApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	return strings.ToLower(result.Label) == "nsfw", nil
}

type batchEntry struct {
	path      string
	mediaType nsfwMediaType
}

func sendBatchToNsfwApi(items []batchEntry) ([]NsfwApiResponse, error) {
	if !(config.NsfwApiEnabled && config.NsfwApiUrl != "") {
		results := make([]NsfwApiResponse, 0, len(items))
		for _, item := range items {
			flag, err := sendFileToFallbackApis(item.path)
			if err != nil {
				return nil, err
			}
			label := "safe"
			if flag {
				label = "nsfw"
			}
			results = append(results, NsfwApiResponse{Label: label})
		}
		return results, nil
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for _, item := range items {
		if err := writer.WriteField("media_types", string(item.mediaType)); err != nil {
			writer.Close()
			return nil, err
		}

		file, err := os.Open(item.path)
		if err != nil {
			writer.Close()
			return nil, fmt.Errorf("open %s: %w", item.path, err)
		}
		part, err := writer.CreateFormFile("files", filepath.Base(item.path))
		if err != nil {
			file.Close()
			writer.Close()
			return nil, fmt.Errorf(
				"create form file for %s: %w",
				item.path,
				err,
			)
		}
		if _, err = io.Copy(part, file); err != nil {
			file.Close()
			writer.Close()
			return nil, fmt.Errorf("copy %s: %w", item.path, err)
		}
		file.Close()
	}
	writer.Close()

	apiURL := strings.TrimRight(config.NsfwApiUrl, "/") + "/detect/batch"
	req, err := http.NewRequest(http.MethodPost, apiURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := httpClient.Do(req)
	if err != nil {
		return fallbackBatchToScanners(items)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fallbackBatchToScanners(items)
	}

	var response struct {
		Results []NsfwApiResponse `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if len(response.Results) != len(items) {
		return fallbackBatchToScanners(items)
	}

	return response.Results, nil
}

func fallbackBatchToScanners(items []batchEntry) ([]NsfwApiResponse, error) {
	results := make([]NsfwApiResponse, 0, len(items))
	for _, item := range items {
		flag, err := sendFileToFallbackApis(item.path)
		if err != nil {
			return nil, err
		}
		label := "safe"
		if flag {
			label = "nsfw"
		}
		results = append(results, NsfwApiResponse{Label: label})
	}
	return results, nil
}

func sendPathBatchToNsfwApi(items []batchEntry) ([]NsfwApiResponse, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for _, item := range items {
		if err := writer.WriteField("media_types", string(item.mediaType)); err != nil {
			writer.Close()
			return nil, err
		}
		if err := writer.WriteField("file_paths", item.path); err != nil {
			writer.Close()
			return nil, err
		}
	}
	writer.Close()

	apiURL := strings.TrimRight(config.NsfwApiUrl, "/") + "/detect/batch"
	req, err := http.NewRequest(http.MethodPost, apiURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := httpClient.Do(req)
	if err != nil {
		return fallbackBatchToScanners(items)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fallbackBatchToScanners(items)
	}

	var response struct {
		Results []NsfwApiResponse `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if len(response.Results) != len(items) {
		return fallbackBatchToScanners(items)
	}

	return response.Results, nil
}

func trackViolationAndMute(client *tg.Client, chatID, userID int64) bool {
	key := nsfwViolationKey{ChatID: chatID, UserID: userID}

	nsfwViolationsMu.Lock()
	count, _ := nsfwViolations.Get(key)
	count++
	nsfwViolations.Set(key, count)
	shouldMute := count >= nsfwViolationThreshold
	if shouldMute {
		nsfwViolations.Set(key, 0)
	}
	nsfwViolationsMu.Unlock()

	if shouldMute {
		muteUntil := int32(time.Now().Add(time.Hour).Unix())
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

func handleNsfwDetected(m *tg.NewMessage) error {
	if _, err := m.Delete(); err != nil {
		return err
	}
	muted := trackViolationAndMute(m.Client, m.ChatID(), m.SenderID())
	mm, _ := m.Respond(buildNsfwWarnMsg(utils.MentionHTML(m.Sender), muted))
	if mm != nil {
		ScheduleDeletion(mm, 30*time.Second)
	}
	return tg.ErrEndGroup
}

func getReplyFileID(m *tg.NewMessage) (string, error) {
	r, err := m.GetReplyMessage()
	if err != nil || r == nil || r.File == nil {
		return "", fmt.Errorf("reply to a media message")
	}
	return r.File.FileID, nil
}

func NsfwMarkSafeCmd(m *tg.NewMessage) error {
	fileID, err := getReplyFileID(m)
	if err != nil {
		_, e := m.Reply("❌ Reply to media to mark as SAFE.")
		return e
	}
	ok, err := database.MarkNsfwForceSafe(m.ChatID(), fileID)
	if err != nil {
		_, e := m.Reply("⚠️ Failed to save safe override.")
		if e != nil {
			return e
		}
		return err
	}
	if !ok {
		_, e := m.Reply("Already marked SAFE.")
		return e
	}
	_, e := m.Reply("✅ Marked SAFE override for this media.")
	return e
}

func NsfwMarkUnsafeCmd(m *tg.NewMessage) error {
	fileID, err := getReplyFileID(m)
	if err != nil {
		_, e := m.Reply("❌ Reply to media to mark as UNSAFE.")
		return e
	}
	ok, err := database.MarkNsfwForceUnsafe(m.ChatID(), fileID)
	if err != nil {
		_, e := m.Reply("⚠️ Failed to save unsafe override.")
		if e != nil {
			return e
		}
		return err
	}
	if !ok {
		_, e := m.Reply("Already marked UNSAFE.")
		return e
	}
	_, e := m.Reply("🚫 Marked UNSAFE override for this media.")
	return e
}

func NsfwResetCmd(m *tg.NewMessage) error {
	fileID, err := getReplyFileID(m)
	if err != nil {
		_, e := m.Reply("❌ Reply to media to reset manual NSFW override.")
		return e
	}
	ok, err := database.UnsetNsfwManual(m.ChatID(), fileID)
	if err != nil {
		_, e := m.Reply("⚠️ Failed to reset override.")
		if e != nil {
			return e
		}
		return err
	}
	if !ok {
		_, e := m.Reply("No manual override found for this media.")
		return e
	}
	_, e := m.Reply("♻️ Manual NSFW override removed.")
	return e
}

func NsfwManualListCmd(m *tg.NewMessage) error {
	if m.IsGroup() {
		btn := tg.NewKeyboard().
			AddRow(tg.Button.URL("📋 View NSFW Manual List", fmt.Sprintf("https://t.me/%s?start=nsfwlist_%d", m.Client.Me().Username, m.ChatID()))).
			Build()
		_, err := m.Reply(
			"Open in PM to view SAFE/UNSAFE manual media list.",
			&tg.SendOptions{ReplyMarkup: btn},
		)
		return err
	}
	if m.IsPrivate() {
		args := strings.Fields(m.Args())
		if len(args) > 0 {
			chatID, err := strconv.ParseInt(args[0], 10, 64)
			if err == nil {
				return HandleNsfwListDeepLink(m, cleanChatID(chatID))
			}
		}
		_, err := m.Reply("Usage: /nsfwlist <chat_id> in PM, or run in group.")
		return err
	}
	return nil
}

func HandleNsfwListDeepLink(m *tg.NewMessage, chatID int64) error {
	safe, unsafe, err := database.GetNsfwManualLists(chatID)
	if err != nil {
		_, e := m.Reply("Failed to fetch NSFW manual list.")
		return e
	}
	if len(safe) == 0 && len(unsafe) == 0 {
		_, e := m.Reply("No manual NSFW overrides saved for this chat.")
		return e
	}
	text := fmt.Sprintf(
		"<b>NSFW Manual Overrides</b>\n\n✅ SAFE: <code>%d</code>\n🚫 UNSAFE: <code>%d</code>",
		len(safe),
		len(unsafe),
	)
	_, err = m.Reply(text, &tg.SendOptions{ParseMode: "HTML"})
	for _, fid := range safe {
		if media, e := tg.ResolveBotFileID(fid); e == nil {
			m.Client.SendMedia(m.Sender.ID, media)
		}
	}
	for _, fid := range unsafe {
		if media, e := tg.ResolveBotFileID(fid); e == nil {
			m.Client.SendMedia(m.Sender.ID, media)
		}
	}
	return nil
}

type NsfwApiResponse struct {
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
}

func sendFileToFallbackApis(filePath string) (bool, error) {
	hfNsfw, hfErr := scanUsingHFAPI(filePath)
	a4Nsfw, a4Err := scanUsingAPI4AI(filePath)

	if hfErr != nil && a4Err != nil {
		return false, fmt.Errorf(
			"fallback scanners failed: hf=%v api4ai=%v",
			hfErr,
			a4Err,
		)
	}

	return hfNsfw || a4Nsfw, nil
}

func scanUsingHFAPI(filePath string) (bool, error) {
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
	_ = writer.Close()

	req, err := http.NewRequest(http.MethodPost, nsfwHFScanAPIURL, body)
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("hf status %d", resp.StatusCode)
	}

	var out struct {
		Scores map[string]float64 `json:"scores"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, err
	}
	p := out.Scores["porn"] * 100
	h := out.Scores["hentai"] * 100
	s := out.Scores["sexy"] * 100
	return (p + h + s) >= nsfwFallbackThreshold, nil
}

func scanUsingAPI4AI(filePath string) (bool, error) {
	if strings.TrimSpace(nsfwAPI4AIKey) == "" {
		return false, fmt.Errorf("api4ai key not configured")
	}
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", filepath.Base(filePath))
	if err != nil {
		return false, err
	}
	if _, err = io.Copy(part, file); err != nil {
		return false, err
	}
	_ = writer.Close()

	req, err := http.NewRequest(http.MethodPost, nsfwAPI4AIURL, body)
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("A4AI-KEY", nsfwAPI4AIKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("api4ai status %d", resp.StatusCode)
	}

	var out struct {
		Results []struct {
			Entities []struct {
				Classes map[string]any `json:"classes"`
			} `json:"entities"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, err
	}
	if len(out.Results) == 0 || len(out.Results[0].Entities) == 0 {
		return false, fmt.Errorf("empty api4ai results")
	}
	nsfw := 0.0
	if v, ok := out.Results[0].Entities[0].Classes["nsfw"]; ok {
		if f, ok := v.(float64); ok {
			nsfw = f * 100
		} else if str, ok := v.(string); ok {
			if ff, err := strconv.ParseFloat(str, 64); err == nil {
				nsfw = ff * 100
			}
		}
	}
	return nsfw >= nsfwFallbackThreshold, nil
}
