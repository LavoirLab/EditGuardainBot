package modules

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"

	"main/config"
	"main/database"
	"main/utils"
)

var hostnameRegex = regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?([a-z0-9.-]+\.[a-z]{2,})`)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "links",
		ModName: "🔗 LinkFilter",
		Help: "🔗 <b>LinkFilter</b> allows admins to restrict messages containing unapproved links.\n\n" +
			"<b>Usage:</b>\n" +
			"➤ <code>/nolinks</code> – See your current link filtering enabled or not.\n" +
			"➤ <code>/nolinks on</code> – Enable link filtering\n" +
			"➤ <code>/nolinks off</code> – Disable link filtering\n" +
			"➤ <code>/allowlink example.com</code> – Allow links from a domain\n" +
			"➤ <code>/removelink example.com</code> – Remove allowed domain\n" +
			"➤ <code>/listlinks</code> – Show all allowed domains\n\n" +
			"🚫 Messages with links not in the allowed list will be automatically deleted.\n" +
			"👮 Only admins can configure this filter.",
		Handlers: []HandlerConfig{
			{
				Pattern: "command:nolinks",
				Handler: RequirePermission("change_info", NoLinksCmd),
			},
			{
				Pattern: "command:allowlink",
				Handler: RequirePermission("change_info", AllowHostCmd),
			},
			{
				Pattern: "command:allowhost",
				Handler: RequirePermission("change_info", AllowHostCmd),
			},
			{
				Pattern: "command:removelink",
				Handler: RequirePermission("change_info", RemoveHostCmd),
			},
			{
				Pattern: "command:listlinks",
				Handler: RequirePermission("change_info", ListAllowedHosts),
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
	if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
		input = "http://" + input
	}
	u, err := url.Parse(input)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

func LinkFilterHandler(m *telegram.NewMessage) error {
	if m.IsAnonymous() {
		return nil
	}

	if isEn, err := database.IsLinkFilterEnabled(m.ChatID()); err != nil || !isEn {
		return nil
	}

	isadmin, _ := utils.IsAdmin(m.Client, m.ChatID(), m.SenderID())
	if isadmin {
		return nil
	}

	var rawLinks []string
	for _, p := range m.Message.Entities {
		if entity, ok := p.(*telegram.MessageEntityURL); ok {
			offset := int(entity.Offset)
			length := int(entity.Length)
			if offset+length > len(m.Text()) {
				continue
			}
			rawLinks = append(rawLinks, m.MessageText()[offset:offset+length])

		}
	}
	if len(rawLinks) == 0 {
		return nil
	}
	allowedHosts, err := database.GetAllowedHostnames(m.ChatID())
	if err != nil {
		return nil
	}
	allowed := make(map[string]struct{}, len(allowedHosts))
	for _, h := range allowedHosts {
		allowed[strings.ToLower(h)] = struct{}{}
	}

	for _, link := range rawLinks {
		host := extractHostname(link)
		if host == "" {
			continue
		}
		if _, ok := allowed[host]; !ok {
			if _, err := m.Delete(); err != nil {
				return err
			}
			mention := utils.MentionHTML(m.Sender)
			_, _ = m.Reply(fmt.Sprintf("🚫 <b>Link Blocked!</b>\n\n%s, <i>unapproved link detected. Only allowed domains are permitted in this group.</i>\n\n💡 <b>Tip:</b> Contact an admin if you think this is a mistake.\n\n🔗 <a href='%s'>Learn how to format links</a>", mention, config.SupportChannel),
				&telegram.SendOptions{
					ParseMode:   "HTML",
					LinkPreview: false,
				},
			)
			return telegram.ErrEndGroup
		}
	}
	return nil
}

func NoLinksCmd(m *telegram.NewMessage) error {
	var err error
	args := strings.Fields(m.Args())
	if _, err = m.Delete(); err != nil {
		return err
	}

	if len(args) < 1 {
		if isEn, err := database.IsLinkFilterEnabled(m.ChatID()); err != nil {
			_, err = m.Reply("⚠️ Usage:\n<code>/nolinks on</code> – Enable link filtering\n<code>/nolinks off</code> – Disable link filtering")
			return err
		} else {
			_, err = m.Reply("Currently Nolinks mode is " + map[bool]string{true: "Enabled", false: "Disabled"}[isEn] + " for your chat.")
			return err
		}
	}
	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err = m.Reply("❌ Invalid option.\nUse <code>/nolinks on</code> or <code>/nolinks off</code>")
		return err
	}
	if err := database.SetLinkFilterEnabled(m.ChatID(), enable); err != nil {
		log.Println("Links.DB error:", err)
		_, err = m.Reply("❌ Failed to update link filter setting.")
		return err
	}
	status := "🔗 Link filter enabled ✅"
	if !enable {
		status = "🔕 Link filter disabled ❌"
	}
	_, err = m.Reply(status)
	return err
}

func AllowHostCmd(m *telegram.NewMessage) error {
	var err error
	args := strings.Fields(m.Args())
	if _, err = m.Delete(); err != nil {
		return err
	}

	if len(args) < 1 {
		_, err = m.Reply("⚠️ Usage: <code>/allowlink github.com</code>")
		return err
	}
	host := extractHostname(args[0])
	if host == "" {
		_, err = m.Reply("❌ Invalid domain or URL.\nExample: <code>/allowlink github.com</code>")
		return err
	}
	if err := database.AddAllowedHostname(m.ChatID(), host); err != nil {
		log.Println("AddAllowedHostname error:", err)
		_, err = m.Reply("❌ Failed to allow host.")
		return err
	}
	_, err = m.Reply("✅ Allowed: <code>" + host + "</code>")
	return err
}

func RemoveHostCmd(m *telegram.NewMessage) error {
	var err error
	args := strings.Fields(m.Args())
	if _, err = m.Delete(); err != nil {
		return err
	}

	if len(args) < 1 {
		_, err = m.Reply("⚠️ Usage: <code>/removelink example.com</code>")
		return err
	}
	host := extractHostname(args[0])
	if host == "" {
		_, err = m.Reply("❌ Invalid domain or URL.\nExample: <code>/removelink github.com</code>")
		return err
	}
	if err := database.RemoveAllowedHostname(m.ChatID(), host); err != nil {
		_, err = m.Reply("❌ Failed to remove host.")
		return err
	}
	_, err = m.Reply("🚫 Removed: <code>" + host + "</code>")
	return err
}

func ListAllowedHosts(m *telegram.NewMessage) error {
	var err error
	if _, err = m.Delete(); err != nil {
		return err
	}
	hosts, err := database.GetAllowedHostnames(m.ChatID())
	if err != nil {
		_, err = m.Reply("❌ Failed to fetch allowed links.")
		return err
	}
	if len(hosts) == 0 {
		_, err = m.Reply("ℹ️ No allowed domains set for this chat.")
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
