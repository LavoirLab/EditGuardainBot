package modules

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/amarnathcjd/gogram/telegram"

	"main/config"
	"main/database"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "log",
		ModName: "Log",
		Handlers: []HandlerConfig{
			{
				Pattern: "participant",
				Handler: botAddded,
			},
		},
	})
}

func botAddded(m *telegram.ParticipantUpdate) error {
	if m.UserID() != BotInfo.ID {
		return nil
	}

	isAdded := m.IsAdded() || m.IsPromoted()
	isRemoved := m.IsKicked() || m.IsBanned()

	groupUsername := "N/A"
	groupTitle := "N/A"
	actor := "N/A"
	chatMemberCount := 0

	if m.Channel != nil {
		if m.Channel.Username != "" {
			groupUsername = "@" + m.Channel.Username
		}
		if m.Channel.Title != "" {
			groupTitle = html.EscapeString(m.Channel.Title)
		}
		chatMemberCount = int(m.Channel.ParticipantsCount)
	}

	if m.Actor != nil {
		if m.Actor.Username != "" {
			actor = "@" + m.Actor.Username
		} else {
			fullName := strings.TrimSpace(m.Actor.FirstName + " " + m.Actor.LastName)
			actor = fmt.Sprintf(`<a href="tg://user?id=%d">%s</a>`, m.ActorID(), html.EscapeString(fullName))
		}
	}

	if _, ok := m.Old.(*telegram.ChannelParticipantAdmin); ok {
		if _, stillMember := m.New.(*telegram.ChannelParticipantObj); stillMember {
			warnMsg := `⚠️ <b>I was demoted from admin!</b>

To work properly, I need admin rights with:
• <code>Delete messages</code>

Leaving... 👋`

			_, _ = m.Client.SendMessage(m.ChannelID(), warnMsg)

			logStr := fmt.Sprintf(
				`⚠️ <b>I was <u>demoted</u> in a group and left.</b>
━━━━━━━━━━━━━━━━━
📌 <b>Group Name:</b> %s
🆔 <b>Group ID:</b> <code>%d</code>
🔗 <b>Username:</b> %s
👤 <b>By:</b> %s
━━━━━━━━━━━━━━━━━`,
				groupTitle,
				m.ChannelID(),
				groupUsername,
				actor,
			)

			_, _ = m.Client.SendMessage(config.LoggerId, logStr)
			database.DeleteServedChat(m.ChannelID())
			return m.Client.LeaveChannel(m.ChannelID())
		}
	}

	if isAdded {
		if admin, ok := m.New.(*telegram.ChannelParticipantAdmin); !ok || !admin.AdminRights.DeleteMessages {
			warnMsg := `⚠️ <b>I was added but lack the required admin rights!</b>

I need:
• <code>Delete messages</code> permission

Leaving... 👋`

			_, _ = m.Client.SendMessage(m.ChannelID(), warnMsg)

			logStr := fmt.Sprintf(
				`⚠️ <b>Bot added without proper permissions</b>
━━━━━━━━━━━━━━━━━
📌 <b>Group Name:</b> %s
🆔 <b>Group ID:</b> <code>%d</code>
🔗 <b>Username:</b> %s
👤 <b>Added By:</b> %s
🚫 <b>Missing:</b> Delete messages
━━━━━━━━━━━━━━━━━`,
				groupTitle,
				m.ChannelID(),
				groupUsername,
				actor,
			)

			_, _ = m.Client.SendMessage(config.LoggerId, logStr)
			database.DeleteServedChat(m.ChannelID())
			return m.Client.LeaveChannel(m.ChannelID())
		}

		_, _ = m.Client.SendMessage(
			m.ChannelID(),
			fmt.Sprintf(
				`Hello 👋 I'm <b>%s</b>, here to help keep the chat transparent and secure.

🚫 I will automatically delete edited messages to maintain clarity.

I'm ready to protect this group! ✅
Let me know if you need any help.`,
				html.EscapeString(BotInfo.FirstName),
			),
		)

		database.AddServedChat(m.ChannelID())

		logStr := fmt.Sprintf(
			`✅ <b>Bot was <u>ADDED</u> to a Group</b>
━━━━━━━━━━━━━━━━━
📌 <b>Group Name:</b> %s
🆔 <b>Group ID:</b> <code>%d</code>
🔗 <b>Username:</b> %s
👤 <b>Added By:</b> %s
👥 <b>Members:</b> %d
━━━━━━━━━━━━━━━━━`,
			groupTitle,
			m.ChannelID(),
			groupUsername,
			actor,
			chatMemberCount,
		)

		_, err := m.Client.SendMessage(config.LoggerId, logStr)
		return err
	}

	if isRemoved {
		logStr := fmt.Sprintf(
			`❌ <b>Bot was <u>KICKED</u> from a Group</b>
━━━━━━━━━━━━━━━━━
📌 <b>Group Name:</b> %s
🆔 <b>Group ID:</b> <code>%d</code>
🔗 <b>Username:</b> %s
👢 <b>Kicked By:</b> %s
🕒 <b>Date:</b> %s
━━━━━━━━━━━━━━━━━`,
			groupTitle,
			m.ChannelID(),
			groupUsername,
			actor,
			time.Unix(int64(m.Date), 0).Format("02 Jan 2006 15:04:05"),
		)

		_, err := m.Client.SendMessage(config.LoggerId, logStr)
		database.DeleteServedChat(m.ChannelID())
		return err
	}

	return nil
}
