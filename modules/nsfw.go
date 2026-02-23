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
	"sync/atomic"
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
	nsfwViolations   = utils.NewCache[string, int](24 * time.Hour)
	nsfwViolationsMu sync.Map
	nsfwSemaphore    = make(chan struct{}, nsfwMaxConcurrent)
	nsfwInflight     int64
)

type nsfwMediaType string

const (
	mediaTypePhoto         nsfwMediaType = "photo"
	mediaTypeStickerStatic nsfwMediaType = "sticker_static"
	mediaTypeStickerVideo  nsfwMediaType = "sticker_video"
	mediaTypeVideo         nsfwMediaType = "video"
)

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
				Pattern: "/nonsfw",
				Handler: RequirePermission("delete_messages", NsfwCmdHandler),
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
		_, err := m.Reply(fmt.Sprintf(
			"🔞 <b>NSFW Detection Status:</b> %s\n\n<i>When enabled, I will scan photos, videos and stickers for adult content.</i>",
			status,
		))
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
	_, err = m.Reply(fmt.Sprintf("🔞 <b>NSFW Detection %s</b>\n\n%s", status, next))
	return err
}

func NsfwHandler(m *tg.NewMessage) error {
	if !config.NsfwApiEnabled || !database.IsNSFWEnabled(m.ChatID()) {
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
		atomic.AddInt64(&nsfwInflight, 1)
		defer atomic.AddInt64(&nsfwInflight, -1)

		nsfwSemaphore <- struct{}{}
		defer func() { <-nsfwSemaphore }()

		isNsfw, err := getOrCheckNsfw(m, mt)
		if err != nil {
			m.Client.Log.Error(fmt.Sprintf("NSFW check error: %v", err))
			return
		}
		if isNsfw {
			handleNsfwDetected(m)
		}
	}()

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

	go func() {
		atomic.AddInt64(&nsfwInflight, 1)
		defer atomic.AddInt64(&nsfwInflight, -1)

		nsfwSemaphore <- struct{}{}
		defer func() { <-nsfwSemaphore }()

		type pendingMedia struct {
			msg       *tg.NewMessage
			path      string
			mediaType nsfwMediaType
		}

		var pending []pendingMedia
		var allPaths []string

		defer func() {
			for _, p := range allPaths {
				os.Remove(p)
			}
		}()

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

			if !config.NsfwOnLocalhost && mt == mediaTypeStickerVideo {
				isNsfw, err := checkStickerVideoNsfw(m)
				if err != nil {
					a.Client.Log.Error(fmt.Sprintf("NSFW sticker video error for msg %d: %v", m.ID, err))
					continue
				}
				if m.File != nil {
					nsfwFileCache.Set(m.File.FileID, isNsfw)
				}
				if isNsfw {
					isNsfwDetected = true
					break
				}
				continue
			}

			if !config.NsfwOnLocalhost && mt == mediaTypeVideo {
				isNsfw, err := extractVideoFramesNsfw(m)
				if err != nil {
					a.Client.Log.Error(fmt.Sprintf("NSFW video frame extraction error for msg %d: %v", m.ID, err))
					continue
				}
				if m.File != nil {
					nsfwFileCache.Set(m.File.FileID, isNsfw)
				}
				if isNsfw {
					isNsfwDetected = true
					break
				}
				continue
			}

			path, err := m.Download()
			if err != nil {
				a.Client.Log.Error(fmt.Sprintf("NSFW album download error for msg %d: %v", m.ID, err))
				continue
			}
			allPaths = append(allPaths, path)
			pending = append(pending, pendingMedia{msg: m, path: path, mediaType: mt})
		}

		if !isNsfwDetected && len(pending) > 0 {
			var items []batchEntry
			for _, p := range pending {
				items = append(items, batchEntry{path: p.path, mediaType: p.mediaType})
			}

			var batchResults []NsfwApiResponse
			var err error
			if config.NsfwOnLocalhost {
				batchResults, err = sendPathBatchToNsfwApi(items)
			} else {
				batchResults, err = sendBatchToNsfwApi(items)
			}
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
		}

		if isNsfwDetected {
			if _, err := a.Delete(); err != nil {
				return
			}
			chatID := a.Messages[0].ChatID()
			userID := a.Messages[0].SenderID()
			muted := trackViolationAndMute(a.Client, chatID, userID)
			a.Messages[0].Respond(buildNsfwWarnMsg(utils.MentionHTML(a.Messages[0].Sender), muted))
		}
	}()

	return nil
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

func isTgsSticker(m *tg.NewMessage) bool {
	doc := m.Sticker()
	if doc == nil {
		return false
	}
	if doc.MimeType == "application/x-tgsticker" {
		return true
	}
	for _, attr := range doc.Attributes {
		if file, ok := attr.(*tg.DocumentAttributeFilename); ok {
			if strings.HasSuffix(strings.ToLower(file.FileName), ".tgs") {
				return true
			}
		}
	}
	return false
}

func getOrCheckNsfw(m *tg.NewMessage, mt nsfwMediaType) (bool, error) {
	cacheKey := ""
	if m.File != nil {
		cacheKey = m.File.FileID
	}

	if cacheKey != "" {
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
		return false, fmt.Errorf("ffmpeg sticker frame extraction failed: %w\n%s", err, out)
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
		"ffmpeg", "-y",
		"-i", videoPath,
		"-vf", "select='not(mod(n\\,trunc(nb_frames/6+1)))',setpts=N/FRAME_RATE/TB",
		"-vsync", "vfr",
		"-frames:v", "6",
		"-q:v", "2",
		framePattern,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("ffmpeg frame extraction failed: %w\n%s", err, out)
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
		return false, fmt.Errorf("API status %d: %s", resp.StatusCode, string(bodyBytes))
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
		return false, fmt.Errorf("API status %d: %s", resp.StatusCode, string(bodyBytes))
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
			return nil, fmt.Errorf("create form file for %s: %w", item.path, err)
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
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var response struct {
		Results []NsfwApiResponse `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if len(response.Results) != len(items) {
		return nil, fmt.Errorf("API returned %d results for %d files", len(response.Results), len(items))
	}

	return response.Results, nil
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
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var response struct {
		Results []NsfwApiResponse `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if len(response.Results) != len(items) {
		return nil, fmt.Errorf("API returned %d results for %d files", len(response.Results), len(items))
	}

	return response.Results, nil
}
	key := fmt.Sprintf("%d_%d", chatID, userID)

	muI, _ := nsfwViolationsMu.LoadOrStore(key, &sync.Mutex{})
	mu := muI.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

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
			client.Log.Error(fmt.Sprintf("NSFW mute failed for user %d in chat %d: %v", userID, chatID, err))
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

func handleNsfwDetected(m *tg.NewMessage) error {
	if _, err := m.Delete(); err != nil {
		return err
	}
	muted := trackViolationAndMute(m.Client, m.ChatID(), m.SenderID())
	if _, err := m.Respond(buildNsfwWarnMsg(utils.MentionHTML(m.Sender), muted)); err != nil {
		return err
	}
	return tg.ErrEndGroup
}

type NsfwApiResponse struct {
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
}
