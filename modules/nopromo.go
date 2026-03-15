package modules

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/amarnathcjd/gogram/telegram"

	"main/database"
	"main/utils"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "nopromo",
		ModName: "🚫 Promo",
		Help: `<emoji id='5929345782560853059'>🚫</emoji> <b>ᴘʀᴏᴍᴏᴛɪᴏɴᴀʟ ᴍᴇssᴀɢᴇ ғɪʟᴛᴇʀ</b> ʙʟᴏᴄᴋs sᴘᴀᴍ ᴀɴᴅ ᴘʀᴏᴍᴏᴛɪᴏɴᴀʟ ᴄᴏɴᴛᴇɴᴛ.

<b>ᴄᴏᴍᴍᴀɴᴅs:</b>
<blockquote>• <code>/nopromo on</code> — ᴇɴᴀʙʟᴇ ᴘʀᴏᴍᴏ ʙʟᴏᴄᴋɪɴɢ <emoji id='6084551628461967966'>✅</emoji>
• <code>/nopromo off</code> — ᴅɪsᴀʙʟᴇ ᴘʀᴏᴍᴏ ʙʟᴏᴄᴋɪɴɢ <emoji id='5465665476971471368'>❌</emoji></blockquote>

<b>ᴅᴇᴛᴇᴄᴛᴇᴅ ᴘᴀᴛᴛᴇʀɴs:</b>
<blockquote>• ᴍᴜʟᴛɪᴘʟᴇ ʀᴇᴘᴇᴀᴛᴇᴅ ʟɪɴᴋs (3+ ᴜʀʟs)
• "ᴊᴏɪɴ ɴᴏᴡ", "ᴄʟɪᴄᴋ ʜᴇʀᴇ" sᴘᴀᴍ
• 24/7 ᴀᴄᴛɪᴠᴇ, ᴠᴄ, ᴄʜᴀᴛ ɢʀᴏᴜᴘ ᴘʀᴏᴍᴏᴛɪᴏɴs
• ᴇxᴄᴇssɪᴠᴇ ᴇᴍᴏᴊɪs (15+ ᴜɴɪǫᴜᴇ ᴇᴍᴏᴊɪs)
• ᴀʟʟ-ᴄᴀᴘs sᴘᴀᴍ ᴍᴇssᴀɢᴇs
• ᴘʀᴏᴍᴏᴛɪᴏɴᴀʟ ᴛᴇxᴛ ᴡɪᴛʜ ᴍᴜʟᴛɪᴘʟᴇ ʟɪɴᴇs
• "ᴍᴀᴋᴇ ɴᴇᴡ ғʀɪᴇɴᴅs", "sᴀғᴇ ғᴏʀ ɢɪʀʟs" ᴘᴀᴛᴛᴇʀɴs</blockquote>

<b><emoji id='5447644880824181073'>⚠️</emoji> ᴛɪᴇʀᴇᴅ ᴀᴄᴛɪᴏɴs:</b>
• sᴄᴏʀᴇ 15-25: ᴡᴀʀɴɪɴɢ ᴍᴇssᴀɢᴇ <emoji id='5447644880824181073'>⚠️</emoji>
• sᴄᴏʀᴇ 25+: ᴍᴇssᴀɢᴇ ᴅᴇʟᴇᴛᴇᴅ <emoji id='5929345782560853059'>🚫</emoji>

<b><emoji id='5420323339723881652'>ℹ️</emoji> ɴᴏᴛᴇ:</b>
<blockquote>– ʜɪɢʜ-sᴄᴏʀᴇ ᴘʀᴏᴍᴏᴛɪᴏɴᴀʟ ᴍᴇssᴀɢᴇs ᴡɪʟʟ ʙᴇ ᴅᴇʟᴇᴛᴇᴅ.
– <emoji id='5370872220149099318'>👮</emoji> ᴏɴʟʏ ɢʀᴏᴜᴘ ᴀᴅᴍɪɴs ᴄᴀɴ ᴄᴏɴғɪɢᴜʀᴇ ᴛʜɪs sᴇᴛᴛɪɴɢ.</blockquote>`,
		Handlers: []HandlerConfig{
			{
				Pattern: "command:nopromo",
				Handler: RequirePermission("delete_messages", NoPromoCmd),
			},
			{
				Pattern: "command:nopromotion",
				Handler: RequirePermission("delete_messages", NoPromoCmd),
			},
		},
	})
}

// Promo detection patterns
var promoPatterns = []struct {
	regex *regexp.Regexp
	score int
}{
	// High priority patterns (25+ points = instant delete)
	{regexp.MustCompile(`(?i)(stop\s*scrolling)`), 15},
	{regexp.MustCompile(`(?i)(you'?ve\s*finally\s*found)`), 12},

	// Medium priority patterns (8-12 points)
	{regexp.MustCompile(`(?i)(join\s*(now|us|today|right\s*now))`), 10},
	{
		regexp.MustCompile(
			`(?i)(24\s*x?\s*7|24/7)\s*(active|chatting|vc|voice)`,
		),
		12,
	},
	{regexp.MustCompile(`(?i)(make\s*new\s*(friends|fantasy))`), 8},
	{regexp.MustCompile(`(?i)(safe\s*for\s*girls)`), 9},
	{regexp.MustCompile(`(?i)(no\s*abuse|respectful\s*environment)`), 7},
	{regexp.MustCompile(`(?i)(voice\s*chat|vc\s*session|song\s*session)`), 8},
	{regexp.MustCompile(`(?i)(chatting\s*(club|group|gc))`), 9},
	{regexp.MustCompile(`(?i)(hurry\s*up|don't\s*miss|limited\s*time)`), 8},
	{regexp.MustCompile(`(?i)(premium\s*(songs?|content))`), 7},
	{regexp.MustCompile(`(?i)(real\s*(people|friendship))`), 6},
	{regexp.MustCompile(`(?i)(late\s*night\s*talks?)`), 6},
	{regexp.MustCompile(`(?i)(feel\s*the\s*(madness|vibe))`), 7},
	{
		regexp.MustCompile(
			`(?i)(looking\s*for\s*(a|the)\s*(best|right)?.*group)`,
		),
		8,
	},
	{regexp.MustCompile(`(?i)(welcome\s*to|join\s*our\s*channel)`), 8},
	{regexp.MustCompile(`(?i)(group\s*owner|group\s*link)`), 7},

	{regexp.MustCompile(`(?i)(जय\s*श्री\s*राम).*(t\.me|join|group)`), 6},
	{regexp.MustCompile(`(?i)(सनातनी).*(group|channel|t\.me)`), 5},
}

// Warning threshold (show warning but don't delete)
const promoWarningThreshold = 15

// Delete threshold (delete message immediately)
const promoDeleteThreshold = 25

func NoPromoCmd(m *telegram.NewMessage) error {
	var err error
	args := strings.Fields(m.Args())
	if _, err = m.Delete(); err != nil {
		return err
	}

	if len(args) < 1 {
		isEnabled, err := database.IsNoPromoEnabled(m.ChatID())
		if err != nil {
			_, err = m.Reply(
				"⚠️ <b>Usage:</b>\n• <code>/nopromo on</code> — Block promotional messages\n• <code>/nopromo off</code> — Allow promotional messages",
			)
			return err
		} else {
			status := "Disabled ❌"
			if isEnabled {
				status = "Enabled ✅"
			}
			_, err = m.Reply(fmt.Sprintf("🚫 <b>Promo Filter Status:</b> %s\n\n<i>When enabled, promotional content like spam links, excessive emojis, and bot promotions will be removed.</i>", status))
			return err
		}
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err = m.Reply(
			"❌ <b>Invalid option.</b>\n\nPlease use <code>/nopromo on</code> or <code>/nopromo off</code>.",
		)
		return err
	}

	if err := database.SetNoPromoEnabled(m.ChatID(), enable); err != nil {
		_, err = m.Reply(
			"⚠️ <b>Update failed.</b>\n\nSomething went wrong while updating the setting. Please try again.",
		)
		return err
	}

	status := "enabled ✅"
	next := "Use <code>/nopromo off</code> to allow promotional messages again."
	if !enable {
		status = "disabled ❌"
		next = "Use <code>/nopromo on</code> to block promotional messages."
	}
	_, err = m.Reply(
		fmt.Sprintf("🚫 <b>Promo Filter %s</b>\n\n%s", status, next),
	)
	return err
}

func NoPromoHandler(m *telegram.NewMessage) error {
	if m.IsReply() || m.IsAnonymous() || m.IsChannelPost() {
		return nil
	}

	isEnabled, err := database.IsNoPromoEnabled(m.ChatID())
	if err != nil || !isEnabled {
		return nil
	}

	if m.Text() == "" {
		return nil
	}

	isadmin, _ := utils.IsAdmin(m.Client, m.ChatID(), m.SenderID())
	if isadmin {
		return nil
	}

	if containsBotUsername(m.Text()) {
		if _, err := m.Delete(); err != nil {
			return err
		}

		mention := utils.MentionHTML(m.Sender)
		_, _ = m.Reply(fmt.Sprintf(
			"🤖 <b>No Bot Promo!</b>\n\n%s, <i>promoting other bots is not allowed in this group.</i> 🚫",
			mention,
		))

		return telegram.ErrEndGroup
	}
	// Calculate promo score
	score := calculatePromoScore(m)

	if score < promoWarningThreshold {
		return nil // Not promotional enough
	}

	mention := utils.MentionHTML(m.Sender)

	// Tiered action based on score
	if score >= promoDeleteThreshold {
		// High score - Delete message
		if _, err := m.Delete(); err != nil {
			return err
		}
		_, _ = m.Reply(
			fmt.Sprintf(
				"🚫 <b>Promotion Detected!</b>\n\n%s, <i>your message was removed because it appears to be promotional content.</i>\n\n📊 <b>Promo Score:</b> <code>%d</code>",
				mention,
				score,
			),
		)
		return telegram.ErrEndGroup
	} else {
		// Medium score - Warning only
		_, _ = m.Reply(fmt.Sprintf("⚠️ <b>Warning: Promo Content!</b>\n\n%s, <i>your message looks highly promotional. Please avoid excessive promotions to stay in the group.</i>\n\n📊 <b>Promo Score:</b> <code>%d</code>", mention, score))
	}

	return nil
}

func calculatePromoScore(m *telegram.NewMessage) int {
	score := 0
	text := m.Text()

	// Check against all patterns
	for _, pattern := range promoPatterns {
		if pattern.regex.MatchString(text) {
			score += pattern.score
		}
	}

	// URL count using entities (already efficient)
	urlCount := 0
	if m.Message.Entities != nil {
		for _, entity := range m.Message.Entities {
			switch entity.(type) {
			case *telegram.MessageEntityURL, *telegram.MessageEntityTextURL:
				urlCount++
			}
		}
	}

	if urlCount >= 5 {
		score += 20
	} else if urlCount >= 3 {
		score += 12
	} else if urlCount >= 2 {
		score += 6
	}

	// Single-pass scanner for lines, emojis, caps, and bullets
	var (
		lineCount   = 1
		emojiCount  = 0
		inEmoji     = false
		upperCount  = 0
		lowerCount  = 0
		letterCount = 0
		bulletCount = 0
	)

	for _, r := range text {
		// Line counting
		if r == '\n' {
			lineCount++
		}

		// Emoji counting
		isEmoji := isEmojiRune(r)
		if isEmoji && !inEmoji {
			emojiCount++
			inEmoji = true
		} else if !isEmoji {
			inEmoji = false
		}

		// Caps detection
		if unicode.IsLetter(r) {
			letterCount++
			if unicode.IsUpper(r) {
				upperCount++
			} else if unicode.IsLower(r) {
				lowerCount++
			}
		}

		// Bullet points and special characters
		if r == '•' || r == '✨' || r == '✅' {
			bulletCount++
		}
	}

	// Line count score
	if lineCount > 10 && len(text) > 500 && urlCount > 0 {
		score += 10
	} else if lineCount > 7 && len(text) > 300 && urlCount > 0 {
		score += 6
	}

	// Emoji score
	if emojiCount > 20 {
		score += 15
	} else if emojiCount > 15 {
		score += 10
	} else if emojiCount > 10 {
		score += 5
	}

	// Caps score
	if len(text) >= 50 && letterCount > 0 {
		capsRatio := float64(upperCount) / float64(letterCount)
		if capsRatio > 0.7 && letterCount > 50 {
			score += 12
		} else if capsRatio > 0.6 && letterCount > 30 {
			score += 8
		} else if capsRatio > 0.5 && letterCount > 50 {
			score += 5
		}
	}

	// Bullet point score
	if bulletCount > 8 {
		score += 10
	} else if bulletCount > 5 {
		score += 6
	}

	return score
}

func isEmojiRune(r rune) bool {
	// Fast-path for standard text (ASCII, Latin, etc.)
	if r < 0x2600 {
		return false
	}

	// Emoji ranges
	return (r >= 0x1F600 && r <= 0x1F64F) || // Emoticons
		(r >= 0x1F300 && r <= 0x1F5FF) || // Misc Symbols and Pictographs
		(r >= 0x1F680 && r <= 0x1F6FF) || // Transport and Map
		(r >= 0x1F1E0 && r <= 0x1F1FF) || // Regional indicators (flags)
		(r >= 0x2600 && r <= 0x26FF) || // Misc symbols
		(r >= 0x2700 && r <= 0x27BF) || // Dingbats
		(r >= 0xFE00 && r <= 0xFE0F) || // Variation Selectors
		(r >= 0x1F900 && r <= 0x1F9FF) || // Supplemental Symbols and Pictographs
		(r >= 0x1FA70 && r <= 0x1FAFF) // Symbols and Pictographs Extended-A
}

func containsBotUsername(text string) bool {
	// A zero-allocation word scanner to replace strings.Fields and strings.ToLower
	n := len(text)
	for i := 0; i < n; {
		// Skip whitespace
		for i < n && unicode.IsSpace(rune(text[i])) {
			i++
		}
		if i >= n {
			break
		}

		// Start of a word
		if text[i] == '@' {
			start := i
			for i < n && !unicode.IsSpace(rune(text[i])) {
				i++
			}
			word := text[start:i]
			// Check for 'bot' suffix case-insensitively
			if len(word) >= 4 { // @ + at least one char + bot
				l := len(word)
				if (word[l-3] == 'b' || word[l-3] == 'B') &&
					(word[l-2] == 'o' || word[l-2] == 'O') &&
					(word[l-1] == 't' || word[l-1] == 'T') {
					return true
				}
			}
		} else {
			// Skip the word
			for i < n && !unicode.IsSpace(rune(text[i])) {
				i++
			}
		}
	}
	return false
}
