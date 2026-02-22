package modules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"

	"main/database"
	"main/utils"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "nohashtags",
		ModName: "# Hashtags",
		Help: "# <b>Hashtag Filter</b> blocks messages containing hashtags.\n\n" +
			"<b>Commands:</b>\n" +
			"• <code>/nohashtags on</code> — Block hashtags ✅\n" +
			"• <code>/nohashtags off</code> — Allow hashtags ❌\n\n" +
			"<b>Detection:</b>\n" +
			"• Any word starting with # symbol\n" +
			"• Example: #join, #promotion, #trending\n\n" +
			"<b>ℹ️ Note:</b> Messages with hashtags will be deleted.\n" +
			"👮 Only group admins can configure this setting.",
		Handlers: []HandlerConfig{
			{
				Pattern: "command:nohashtags",
				Handler: RequirePermission("change_info", NoHashtagsCmd),
			},
			{
				Pattern: "command:nohashtag",
				Handler: RequirePermission("change_info", NoHashtagsCmd),
			},
		},
	})
}

var hashtagRegex = regexp.MustCompile(`#\w+`)

func NoHashtagsCmd(m *telegram.NewMessage) error {
	var err error
	args := strings.Fields(m.Args())
	if _, err = m.Delete(); err != nil {
		return err
	}

	if len(args) < 1 {
		isEnabled, err := database.IsNoHashtagsEnabled(m.ChatID())
		if err != nil {
			_, err = m.Reply(
				"⚠️ <b>Usage:</b>\n• <code>/nohashtags on</code> — Block hashtags\n• <code>/nohashtags off</code> — Allow hashtags",
			)
			return err
		} else {
			status := "Disabled ❌"
			if isEnabled {
				status = "Enabled ✅"
			}
			_, err = m.Reply(fmt.Sprintf("🏷️ <b>Hashtag Filter Status:</b> %s\n\n<i>When enabled, messages containing hashtags (#example) will be automatically removed.</i>", status))
			return err
		}
	}

	enable, err := utils.ParseBool(args[0])
	if err != nil {
		_, err = m.Reply(
			"❌ <b>Invalid option.</b>\n\nPlease use <code>/nohashtags on</code> or <code>/nohashtags off</code>.",
		)
		return err
	}

	if err := database.SetNoHashtagsEnabled(m.ChatID(), enable); err != nil {
		_, err = m.Reply(
			"⚠️ <b>Update failed.</b>\n\nSomething went wrong while updating the setting. Please try again.",
		)
		return err
	}

	status := "enabled ✅"
	next := "Use <code>/nohashtags off</code> to allow hashtags again."
	if !enable {
		status = "disabled ❌"
		next = "Use <code>/nohashtags on</code> to block hashtags."
	}
	_, err = m.Reply(
		fmt.Sprintf("🏷️ <b>Hashtag filter %s</b>\n\n%s", status, next),
	)
	return err
}

func NoHashtagsHandler(m *telegram.NewMessage) error {
	if m.IsAnonymous() {
		return nil
	}
	isEnabled, err := database.IsNoHashtagsEnabled(m.ChatID())
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

	// Check for hashtags
	if !hashtagRegex.MatchString(m.Text()) {
		return nil
	}

	// Delete message with hashtags
	if _, err := m.Delete(); err != nil {
		return err
	}

	mention := utils.MentionHTML(m.Sender)
	_, _ = m.Reply(
		fmt.Sprintf(
			"🏷️ <b>No Hashtags!</b>\n\n%s, <i>hashtags are not permitted in this group to prevent spamming and maintain clarity.</i> 🚫",
			mention,
		),
	)
	return telegram.ErrEndGroup
}
