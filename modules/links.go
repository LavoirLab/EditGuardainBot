package modules

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"

	"main/config"
	"main/database"
	"main/utils"
)

var hostnameRegex = regexp.MustCompile(
	`(?i)(?:https?://)?(?:www\.)?([a-z0-9.-]+\.[a-z]{2,})`,
)

// allowedHostsCache stores the pre-built map of allowed hostnames per chat.
// This eliminates the need to rebuild the map and query the DB for every message.
var allowedHostsCache = utils.NewCache[int64, map[string]struct{}](time.Hour)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "links",
		ModName: "🔗 Links",
		Help: `🔗 <b>Lɪɴᴋғɪʟᴛᴇʀ</b> ᴀʟʟᴏᴡs ᴀᴅᴍɪɴs ᴛᴏ ʀᴇsᴛʀɪᴄᴛ ᴍᴇssᴀɢᴇs ᴄᴏɴᴛᴀɪɴɪɴɢ ᴜɴᴀᴘᴘʀᴏᴠᴇᴅ ʟɪɴᴋs.

<b>ᴜsᴀɢᴇ:</b>
<blockquote>➤ <code>/nolinks</code> – sᴇᴇ ʏᴏᴜʀ ᴄᴜʀʀᴇɴᴛ ʟɪɴᴋ ғɪʟᴛᴇʀɪɴɢ ᴇɴᴀʙʟᴇᴅ ᴏʀ ɴᴏᴛ.
➤ <code>/nolinks on</code> – ᴇɴᴀʙʟᴇ ʟɪɴᴋ ғɪʟᴛᴇʀɪɴɢ
➤ <code>/nolinks off</code> – ᴅɪsᴀʙʟᴇ ʟɪɴᴋ ғɪʟᴛᴇʀɪɴɢ
➤ <code>/allowlink example.com</code> – ᴀʟʟᴏᴡ ʟɪɴᴋs ғʀᴏᴍ ᴀ ᴅᴏᴍᴀɪɴ
➤ <code>/removelink example.com</code> – ʀᴇᴍᴏᴠᴇ ᴀʟʟᴏᴡᴇᴅ ᴅᴏᴍᴀɪɴ
➤ <code>/listlinks</code> – sʜᴏᴡ ᴀʟʟ ᴀʟʟᴏᴡᴇᴅ ᴅᴏᴍᴀɪɴs</blockquote>

<blockquote><emoji id='5929345782560853059'>🚫</emoji> ᴍᴇssᴀɢᴇs ᴡɪᴛʜ ʟɪɴᴋs ɴᴏᴛ ɪɴ ᴛʜᴇ ᴀʟʟᴏᴡᴇᴅ ʟɪsᴛ ᴡɪʟʟ ʙᴇ ᴀᴜᴛᴏᴍᴀᴛɪᴄᴀʟʟʏ ᴅᴇʟᴇᴛᴇᴅ.
<emoji id='5370872220149099318'>👮</emoji> ᴏɴʟʏ ᴀᴅᴍɪɴs ᴄᴀɴ ᴄᴏɴғɪɢᴜʀᴇ ᴛʜɪs ғɪʟᴛᴇʀ.</blockquote>`,
		Handlers: []HandlerConfig{
			{
				Pattern: "command:nolinks",
				Handler: RequirePermission("delete_messages", NoLinksCmd),
			},
			{
				Pattern: "command:allowlink",
				Handler: RequirePermission("delete_messages", AllowHostCmd),
			},
			{
				Pattern: "command:allowhost",
				Handler: RequirePermission("delete_messages", AllowHostCmd),
			},
			{
				Pattern: "command:removelink",
				Handler: RequirePermission("delete_messages", RemoveHostCmd),
			},
			{
				Pattern: "command:listlinks",
				Handler: RequirePermission("delete_messages", ListAllowedHosts),
			},
		},
	})
}

func extractHostname(input string) string {
	input = strings.TrimSpace(input)
	matches := hostnameRegex.FindStringSubmatch(input)
	if len(matches) >= 2 {
		return strings.ToLower(matches[1])
	}
	if !strings.HasPrefix(input, "http://") &&
		!strings.HasPrefix(input, "https://") {
		input = "http://" + input
	}
	u, err := url.Parse(input)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

func LinkFilterHandler(m *telegram.NewMessage) error {
	// Fast path: check for entities before any other processing
	if len(m.Message.Entities) == 0 || m.IsAnonymous() || m.IsChannelPost() {
		return nil
	}

	if isEn, err := database.IsLinkFilterEnabled(m.ChatID()); err != nil ||
		!isEn {
		return nil
	}

	isadmin, _ := utils.IsAdmin(m.Client, m.ChatID(), m.SenderID())
	if isadmin {
		return nil
	}

	// Optimization: Fetch the allowed map from cache.
	// This reduces allocations by avoiding rebuilds of the same map.
	allowed, ok := allowedHostsCache.Get(m.ChatID())
	if !ok {
		allowedHosts, err := database.GetAllowedHostnames(m.ChatID())
		if err != nil {
			return nil
		}
		allowed = make(map[string]struct{}, len(allowedHosts))
		for _, h := range allowedHosts {
			allowed[strings.ToLower(h)] = struct{}{}
		}
		allowedHostsCache.Set(m.ChatID(), allowed)
	}

	// Optimization: Iterate through entities once and process links directly,
	// removing the need for an intermediate slice of link strings.
	for _, p := range m.Message.Entities {
		if entity, ok := p.(*telegram.MessageEntityURL); ok {
			offset := int(entity.Offset)
			length := int(entity.Length)
			if offset+length > len(m.Text()) {
				continue
			}
			link := m.MessageText()[offset : offset+length]
			host := extractHostname(link)
			if host == "" {
				continue
			}

			if _, ok := allowed[host]; !ok {
				if _, err := m.Delete(); err != nil {
					return err
				}
				mention := utils.MentionHTML(m.Sender)
				mm, _ := m.Reply(
					fmt.Sprintf(
						"🚫 <b>Link Blocked!</b>\n\n%s, <i>unapproved link detected. Only allowed domains are permitted in this group.</i>\n\n💡 <b>Tip:</b> Contact an admin if you think this is a mistake.\n\n🔗 <a href='%s'>Learn how to format links</a>",
						mention,
						config.SupportChannel,
					),
					&telegram.SendOptions{
						ParseMode:   "HTML",
						LinkPreview: false,
					},
				)
				if mm != nil {
					ScheduleDeletion(mm, 30*time.Second)
				}
				return telegram.ErrEndGroup
			}
		}
	}

	return nil
}

func NoLinksCmd(m *telegram.NewMessage) error {
	args := strings.Fields(m.Args())
	if _, err := m.Delete(); err != nil {
		return err
	}

	if len(args) < 1 {
		isEn, err := database.IsLinkFilterEnabled(m.ChatID())
		if err != nil {
			_, err := m.Reply(
				"⚠️ <b>Usage:</b>\n• <code>/nolinks on</code> — Enable link filtering\n• <code>/nolinks off</code> — Disable link filtering",
			)
			return err
		}
		status := "Disabled ❌"
		if isEn {
			status = "Enabled ✅"
		}
		_, err = m.Reply("🔗 <b>Link Filtering Status:</b> " + status + "\n\n<i>When enabled, only links from approved domains will be allowed. Use <code>/nolinks on</code> to enable.</i>")
		return err
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err := m.Reply(
			"❌ <b>Invalid option.</b>\n\nPlease use:\n• <code>/nolinks on</code> — To enable\n• <code>/nolinks off</code> — To disable",
		)
		return err
	}

	if err := database.SetLinkFilterEnabled(m.ChatID(), enable); err != nil {
		log.Println("Links.DB error:", err)
		_, err := m.Reply(
			"⚠️ <b>Oops! Failed to update settings.</b>\n\nSomething went wrong while updating the link filter. Please try again.",
		)
		return err
	}
	allowedHostsCache.Delete(m.ChatID())

	status := "enabled ✅"
	next := "Use <code>/nolinks off</code> to disable it."
	if !enable {
		status = "disabled ❌"
		next = "Use <code>/nolinks on</code> to enable it again."
	}
	_, err = m.Reply(
		fmt.Sprintf("🔗 <b>Link filtering %s</b>\n\n%s", status, next),
	)
	return err
}

func AllowHostCmd(m *telegram.NewMessage) error {
	args := strings.Fields(m.Args())
	if _, err := m.Delete(); err != nil {
		return err
	}

	if len(args) < 1 {
		_, err := m.Reply(
			"⚠️ <b>Usage:</b> <code>/allowlink &lt;domain&gt;</code>\n\n<b>Example:</b>\n• <code>/allowlink google.com</code>\n• <code>/allowlink github.com</code>",
		)
		return err
	}
	host := extractHostname(args[0])
	if host == "" {
		_, err := m.Reply(
			"❌ <b>Invalid domain or URL.</b>\n\nPlease provide a valid domain.\n<b>Example:</b> <code>/allowlink github.com</code>",
		)
		return err
	}
	if err := database.AddAllowedHostname(m.ChatID(), host); err != nil {
		log.Println("AddAllowedHostname error:", err)
		_, err := m.Reply(
			"⚠️ <b>Failed to add domain.</b>\n\nSomething went wrong while adding the domain to the allow-list. Please try again.",
		)
		return err
	}
	allowedHostsCache.Delete(m.ChatID())
	_, err := m.Reply(
		fmt.Sprintf(
			"✅ <b>Domain Allowed:</b> <code>%s</code>\n\nLinks from this domain will no longer be blocked. Use <code>/removelink %s</code> to remove it.",
			host,
			host,
		),
	)
	return err
}

func RemoveHostCmd(m *telegram.NewMessage) error {
	args := strings.Fields(m.Args())
	if _, err := m.Delete(); err != nil {
		return err
	}

	if len(args) < 1 {
		_, err := m.Reply(
			"⚠️ <b>Usage:</b> <code>/removelink &lt;domain&gt;</code>\n\n<b>Example:</b> <code>/removelink github.com</code>",
		)
		return err
	}
	host := extractHostname(args[0])
	if host == "" {
		_, err := m.Reply(
			"❌ <b>Invalid domain or URL.</b>\n\nPlease provide a valid domain to remove.\n<b>Example:</b> <code>/removelink github.com</code>",
		)
		return err
	}
	if err := database.RemoveAllowedHostname(m.ChatID(), host); err != nil {
		_, err := m.Reply(
			"⚠️ <b>Failed to remove domain.</b>\n\nSomething went wrong while removing the domain from the allow-list. Please try again.",
		)
		return err
	}
	allowedHostsCache.Delete(m.ChatID())
	_, err := m.Reply(
		fmt.Sprintf(
			"🚫 <b>Domain Removed:</b> <code>%s</code>\n\nLinks from this domain will now be filtered again. Use <code>/allowlink %s</code> to add it back.",
			host,
			host,
		),
	)
	return err
}

func ListAllowedHosts(m *telegram.NewMessage) error {
	if _, err := m.Delete(); err != nil {
		return err
	}
	hosts, err := database.GetAllowedHostnames(m.ChatID())
	if err != nil {
		_, err := m.Reply("❌ Failed to fetch allowed links.")
		return err
	}
	if len(hosts) == 0 {
		_, err := m.Reply(
			"ℹ️ <b>No allowed domains yet.</b>\n\nAll external links are currently being filtered. Use <code>/allowlink &lt;domain&gt;</code> to add your first exception!",
		)
		return err
	}
	var text strings.Builder
	text.WriteString("✅ <b>Allowed Domains:</b>\n")
	for _, h := range hosts {
		text.WriteString("• <code>" + h + "</code>\n")
	}
	_, err = m.Reply(text.String())
	return err
}
