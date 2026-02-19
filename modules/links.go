package modules

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"

	"main/config"
	"main/utils"
	"main/database"
)

var hostnameRegex = regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?([a-z0-9.-]+\.[a-z]{2,})`)

func init() {
	AddHelp(
		"🔗 LinkFilter",
		"linkfilter_help",
		"🔗 <b>LinkFilter</b> allows admins to restrict messages containing unapproved links.\n\n"+
			"<b>Usage:</b>\n"+
			"➤ <code>/nolinks/code> – See your current link filtering enabled or not.\n"+
			"➤ <code>/nolinks on</code> – Enable link filtering\n"+
			"➤ <code>/nolinks off</code> – Disable link filtering\n"+
			"➤ <code>/allowlink example.com</code> – Allow links from a domain\n"+
			"➤ <code>/removelink example.com</code> – Remove allowed domain\n"+
			"➤ <code>/listlinks</code> – Show all allowed domains\n\n"+
			"🚫 Messages with links not in the allowed list will be automatically deleted.\n"+
			"👮 Only admins can configure this filter.",
	)
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

func deleteLinkMessage(m *telegram.NewMessage) error {
	if !IsSupergroup(m) {
		return nil
	}

	if isEn, err := database.IsLinkFilterEnabled(m.ChatID()); err != nil || !isEn {
		return nil
	}
	if ShouldIgnoreGroupAnonymous(m) {
		return nil
	}
	if isadmin, err := utils.IsChatAdmin(m.Client, m.ChatID(), m.Sender.ID); err != nil {
		L(m, "Modules -> nolinks -> utils.IsChatAdmin()", err)

		return nil
	} else if isadmin {
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
		return L(m, "Modules -> nolinks -> database.GetAllowedHostnames", err)
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
			if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
				return telegram.EndGroup
			}
			m.Respond(fmt.Sprintf("⚠️ Unapproved link detected. Only allowed domains are permitted.\nIf this is a mistake, please contact an admin.\n\nOr use <a href='%s'>Example formatted link</a>", config.SupportChannel),
				telegram.SendOptions{
					ParseMode:   "HTML",
					LinkPreview: false,
				},
			)
			return telegram.EndGroup
		}
	}
	return nil
}

func NoLinksCmd(m *telegram.NewMessage) error {
	args := strings.Fields(m.Text())
	if isgroup := IsValidSupergroup(m); !isgroup {
		return telegram.EndGroup
	}
	if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
		return telegram.EndGroup
	} else if err != nil {
		return L(m, "Modules -> nolinks -> m.Delete()", err)
	}
	if isadmin, err := utils.IsChatAdmin(m.Client, m.ChatID(), m.SenderID()); err != nil {
		return L(m, "Modules -> nolinks -> utils.IsChatAdmin()", err)
	} else if !isadmin {
		m.Respond("Access denied: Only group admins can use this command.")

		return telegram.EndGroup
	}

	if len(args) < 2 {

		if isEn, err := database.IsLinkFilterEnabled(m.ChatID()); err != nil {
			m.Respond("⚠️ Usage:\n<code>/nolinks on</code> – Enable link filtering\n<code>/nolinks off</code> – Disable link filtering")
		} else {
			m.Respond("Currently Nolinks mode is " + map[bool]string{true: "Enabled", false: "Disabled"}[isEn] + " for your chat.")
		}
		return telegram.EndGroup
	}
	arg := strings.ToLower(args[1])
	enable := arg == "on"
	if arg != "on" && arg != "off" {
		m.Respond("❌ Invalid option.\nUse <code>/nolinks on</code> or <code>/nolinks off</code>")
		return telegram.EndGroup
	}
	if err := database.SetLinkFilterEnabled(m.ChatID(), enable); err != nil {
		log.Println("Links.DB error:", err)
		m.Respond("❌ Failed to update link filter setting.")
		return telegram.EndGroup
	}
	status := "🔗 Link filter enabled ✅"
	if !enable {
		status = "🔕 Link filter disabled ❌"
	}
	m.Respond(status)
	return telegram.EndGroup
}

func AllowHostCmd(m *telegram.NewMessage) error {
	args := strings.Fields(m.MessageText())

	if isgroup := IsValidSupergroup(m); !isgroup {
		return telegram.EndGroup
	}

	if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
		return telegram.EndGroup
	} else if err != nil {
		return L(m, "Modules -> nolinks -> m.Delete()", err)
	}
	if isadmin, err := utils.IsChatAdmin(m.Client, m.ChatID(), m.SenderID()); err != nil {
		return L(m, "Modules -> nolinks -> utils.IsChatAdmin()", err)
	} else if !isadmin {
		m.Respond("Access denied: Only group admins can use this command.")

		return telegram.EndGroup
	}

	if len(args) < 2 {
		m.Respond("⚠️ Usage: <code>/allowlink github.com</code>")
		return telegram.EndGroup
	}
	host := extractHostname(args[1])
	if host == "" {
		m.Respond("❌ Invalid domain or URL.\nExample: <code>/allowlink github.com</code>")
		return telegram.EndGroup
	}
	if err := database.AddAllowedHostname(m.ChatID(), host); err != nil {
		log.Println("AddAllowedHostname error:", err)
		m.Respond("❌ Failed to allow host.")
		return telegram.EndGroup
	}
	m.Respond("✅ Allowed: <code>" + host + "</code>")
	return telegram.EndGroup
}

func RemoveHostCmd(m *telegram.NewMessage) error {
	args := strings.Fields(m.MessageText())

	if isgroup := IsValidSupergroup(m); !isgroup {
		return nil
	}
	if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
		return telegram.EndGroup
	} else if err != nil {
		return L(m, "Modules -> nolinks -> m.Delete()", err)
	}

	if isadmin, err := utils.IsChatAdmin(m.Client, m.ChatID(), m.SenderID()); err != nil {
		return L(m, "Modules -> nolinks -> utils.IsChatAdmin()", err)
	} else if !isadmin {
		m.Respond("Access denied: Only group admins can use this command.")

		return telegram.EndGroup
	}

	if len(args) < 2 {
		m.Respond("⚠️ Usage: <code>/removelink example.com</code>")

		return telegram.EndGroup
	}
	host := extractHostname(args[1])
	if host == "" {
		m.Respond("❌ Invalid domain or URL.\nExample: <code>/removelink github.com</code>")
		return telegram.EndGroup
	}
	if err := database.RemoveAllowedHostname(m.ChatID(), host); err != nil {
		L(m, "Modules -> Nolinks -> database.RemoveAllowedHostname", err)

		m.Respond("❌ Failed to remove host.")
		return telegram.EndGroup
	}
	m.Respond("🚫 Removed: <code>" + host + "</code>")
	return telegram.EndGroup
}

func ListAllowedHosts(m *telegram.NewMessage) error {
	if isgroup := IsValidSupergroup(m); !isgroup {
		return telegram.EndGroup
	}
	if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
		return telegram.EndGroup
	} else if err != nil {
		return L(m, "Modules -> nolinks -> m.Delete()", err)
	}
	hosts, err := database.GetAllowedHostnames(m.ChatID())
	if err != nil {
		L(m, "Modules -> Nolinks -> database.GetAllowedHostname", err)
		m.Respond("❌ Failed to fetch allowed links.")
		return telegram.EndGroup
	}
	if len(hosts) == 0 {
		m.Respond("ℹ️ No allowed domains set for this chat.")
		return telegram.EndGroup
	}
	var text strings.Builder
	text.WriteString("✅ <b>Allowed Domains:</b>\n")
	for _, h := range hosts {
		text.WriteString("• <code>" + h + "</code>\n")
	}
	m.Respond(text.String())
	return telegram.EndGroup
}
