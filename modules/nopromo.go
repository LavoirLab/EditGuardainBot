// file: modules/nopromo.go
package modules

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode"

	"github.com/amarnathcjd/gogram/telegram"

	"main/config/helpers"
	"main/database"
)

func init() {
	AddHelp(
		"🚫 NoPromo",
		"nopromo_help",
		"🚫 <b>Promotional Message Filter</b> blocks spam and promotional content.\n\n"+
			"<b>Commands:</b>\n"+
			"• <code>/nopromo on</code> — Enable promo blocking ✅\n"+
			"• <code>/nopromo off</code> — Disable promo blocking ❌\n\n"+
			"<b>Detected Patterns:</b>\n"+
			"• Multiple repeated links (3+ URLs)\n"+
			"• \"Join now\", \"Click here\" spam\n"+
			"• 24/7 active, VC, chat group promotions\n"+
			"• Excessive emojis (15+ unique emojis)\n"+
			"• ALL-CAPS spam messages\n"+
			"• Promotional text with multiple lines\n"+
			"• \"Make new friends\", \"Safe for girls\" patterns\n\n"+
			"<b>⚠️ Tiered Actions:</b>\n"+
			"• Score 15-25: Warning message ⚠️\n"+
			"• Score 25+: Message deleted 🚫\n\n"+
			"<b>ℹ️ Note:</b> High-score promotional messages will be deleted.\n"+
			"👮 Only group admins can configure this setting.",
	)
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
	{regexp.MustCompile(`(?i)(24\s*x?\s*7|24/7)\s*(active|chatting|vc|voice)`), 12},
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
	{regexp.MustCompile(`(?i)(looking\s*for\s*(a|the)\s*(best|right)?.*group)`), 8},
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
	args := strings.Fields(m.Text())
	if isgroup := IsValidSupergroup(m); !isgroup {
		return telegram.EndGroup
	}
	if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
		return telegram.EndGroup
	} else if err != nil {
		return L(m, "Modules -> nopromo -> m.Delete()", err)
	}
	if isadmin, err := helpers.IsChatAdmin(m.Client, m.ChatID(), m.SenderID()); err != nil {
		return L(m, "Modules -> nopromo -> helpers.IsChatAdmin()", err)
	} else if !isadmin {
		m.Respond("Access denied: Only group admins can use this command.")
		return telegram.EndGroup
	}

	if len(args) < 2 {
		isEnabled, err := database.IsNoPromoEnabled(m.ChatID())
		if err != nil {
			m.Respond("⚠️ Usage:\n<code>/nopromo on</code> — Enable\n<code>/nopromo off</code> — Disable")
		} else {
			status := map[bool]string{true: "Enabled", false: "Disabled"}[isEnabled]
			m.Respond(fmt.Sprintf("Currently NoPromo is %s for your chat.", status))
		}
		return telegram.EndGroup
	}

	arg := strings.ToLower(args[1])
	enable := arg == "on"
	if arg != "on" && arg != "off" {
		m.Respond("❌ Invalid option.\nUse <code>/nopromo on</code> or <code>/nopromo off</code>")
		return telegram.EndGroup
	}

	if err := database.SetNoPromoEnabled(m.ChatID(), enable); err != nil {
		m.Respond("❌ Failed to update setting.")
		return L(m, "Modules -> nopromo -> SetNoPromoEnabled", err)
	}

	status := "🚫 Promotional message blocking enabled ✅"
	if !enable {
		status = "✅ Promotional messages allowed ❌"
	}
	m.Respond(status)
	return telegram.EndGroup
}

func handlePromoMessages(m *telegram.NewMessage) error {
	if !IsSupergroup(m) || m.IsReply() {
		return nil
	}

	isEnabled, err := database.IsNoPromoEnabled(m.ChatID())
	if err != nil || !isEnabled {
		return nil
	}

	if m.Text() == "" {
		return nil
	}

	if ShouldIgnoreGroupAnonymous(m) {
		return nil
	}

	isadmin, err := helpers.IsChatAdmin(m.Client, m.ChatID(), m.Sender.ID)
	if err != nil {
		L(m, "Modules -> nopromo -> helpers.IsChatAdmin()", err)
		return nil
	}
	if isadmin {
		return nil
	}

if containsBotUsername(m.Text()) {

        if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
                return telegram.EndGroup
        } else if err != nil {
                return L(m, "Modules -> nopromo -> bot mention delete", err)
        }

        var mention string
        if m.Sender.Username != "" {
                mention = "@" + m.Sender.Username
        } else {
                mention = fmt.Sprintf(
                        "<a href='tg://user?id=%d'>%s</a>",
                        m.Sender.ID,
                        html.EscapeString(m.Sender.FirstName),
                )
        }

        m.Respond(fmt.Sprintf(
                "🚫 %s, promoting bot usernames is not allowed in this group.",
                mention,
        ))

        return telegram.EndGroup
}
	// Calculate promo score
	score := calculatePromoScore(m)

	if score < promoWarningThreshold {
		return nil // Not promotional enough
	}

	var mention string
	if m.Sender.Username != "" {
		mention = "@" + m.Sender.Username
	} else {
		mention = fmt.Sprintf("<a href='tg://user?id=%d'>%s</a>", m.Sender.ID, html.EscapeString(m.Sender.FirstName))
	}

	// Tiered action based on score
	if score >= promoDeleteThreshold {
		// High score - Delete message
		if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
			return telegram.EndGroup
		} else if err != nil {
			return L(m, "Modules -> nopromo -> m.Delete()", err)
		}
		m.Respond(fmt.Sprintf("🚫 %s, promotional messages are not allowed in this group. (Score: %d)", mention, score))
	} else {
		// Medium score - Warning only
		m.Respond(fmt.Sprintf("⚠️ %s, your message looks promotional. Please avoid excessive promotional content. (Score: %d)", mention, score), telegram.SendOptions{
			ReplyID: m.ID,
		})
	}

	return telegram.EndGroup
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

	// Count URLs using Telegram entities (proper way)
	urlCount := 0
	if m.Message.Entities != nil {
		for _, entity := range m.Message.Entities {
			switch entity.(type) {
			case *telegram.MessageEntityURL:
				urlCount++
			case *telegram.MessageEntityTextURL:
				urlCount++
			}
		}
	}

	// Score based on URL count
	if urlCount >= 5 {
		score += 20 // Many URLs = likely spam
	} else if urlCount >= 3 {
		score += 12
	} else if urlCount >= 2 {
		score += 6
	}

	// Long messages with multiple lines (likely promo)
	lines := strings.Split(text, "\n")
	if len(lines) > 10 && len(text) > 500 && urlCount > 0 {
		score += 10
	} else if len(lines) > 7 && len(text) > 300 && urlCount > 0 {
		score += 6
	}

	// Proper emoji counting
	emojiCount := countEmojis(text)
	if emojiCount > 20 {
		score += 15
	} else if emojiCount > 15 {
		score += 10
	} else if emojiCount > 10 {
		score += 5
	}

	// ALL-CAPS spam detection
	capsScore := detectAllCaps(text)
	score += capsScore

	// Check for promotional structure (bullet points, formatting)
	bulletCount := strings.Count(text, "•") +
		strings.Count(text, "✨") +
		strings.Count(text, "✅")

	if bulletCount > 8 {
		score += 10
	} else if bulletCount > 5 {
		score += 6
	}

	return score
}

// Proper emoji detection using Unicode properties
func countEmojis(text string) int {
	count := 0
	inEmoji := false

	for _, r := range text {
		isEmoji := isEmojiRune(r)

		if isEmoji && !inEmoji {
			count++
			inEmoji = true
		} else if !isEmoji {
			inEmoji = false
		}
	}

	return count
}

func isEmojiRune(r rune) bool {
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

// Detect excessive ALL-CAPS spam
func detectAllCaps(text string) int {
	if len(text) < 50 {
		return 0 // Too short to judge
	}

	upperCount := 0
	lowerCount := 0
	letterCount := 0

	for _, r := range text {
		if unicode.IsLetter(r) {
			letterCount++
			if unicode.IsUpper(r) {
				upperCount++
			} else if unicode.IsLower(r) {
				lowerCount++
			}
		}
	}

	if letterCount == 0 {
		return 0
	}

	capsRatio := float64(upperCount) / float64(letterCount)

	// High caps ratio with significant text = spam
	if capsRatio > 0.7 && letterCount > 50 {
		return 12
	} else if capsRatio > 0.6 && letterCount > 30 {
		return 8
	} else if capsRatio > 0.5 && letterCount > 50 {
		return 5
	}

	return 0
}

func containsBotUsername(text string) bool {
        words := strings.Fields(text)
        for _, w := range words {
                if strings.HasPrefix(w, "@") {
                        uname := strings.ToLower(strings.TrimPrefix(w, "@"))
                        if strings.HasSuffix(uname, "bot") {
                                return true
                        }
                }
        }
        return false
}