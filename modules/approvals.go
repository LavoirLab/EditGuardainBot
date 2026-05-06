package modules

import (
	"fmt"

	"github.com/amarnathcjd/gogram/telegram"

	"main/database"
	"main/utils"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "approvals",
		ModName: "✅ Approvals",
		Help:    "<b>Approval commands:</b>\n<blockquote>• <code>/approve</code> (reply/mention/id)\n• <code>/unapprove</code> or <code>/disapprove</code>\n• <code>/approved</code> to check status</blockquote>\n\nApproved users are ignored by automated moderation actions.",
		Handlers: []HandlerConfig{
			{
				Pattern: "command:approve",
				Handler: RequirePermission("delete_messages", approveCmd),
			},
			{
				Pattern: "command:unapprove",
				Handler: RequirePermission("delete_messages", unapproveCmd),
			},
			{
				Pattern: "command:disapprove",
				Handler: RequirePermission("delete_messages", unapproveCmd),
			},
			{
				Pattern: "command:approved",
				Handler: RequirePermission("delete_messages", approvedCmd),
			},
		},
	})
}

func approveCmd(m *telegram.NewMessage) error {
	userID := utils.ExtractUser(m)
	if userID == 0 {
		return telegram.ErrEndGroup
	}
	if err := database.ApproveUser(m.ChatID(), userID); err != nil {
		_, e := m.Reply("⚠️ Failed to approve user.")
		if e != nil {
			return e
		}
		return err
	}
	_, err := m.Reply(
		fmt.Sprintf(
			"✅ User <code>%d</code> is now approved and exempt from automated actions.",
			userID,
		),
		&telegram.SendOptions{ParseMode: "HTML"},
	)
	return err
}

func unapproveCmd(m *telegram.NewMessage) error {
	userID := utils.ExtractUser(m)
	if userID == 0 {
		return telegram.ErrEndGroup
	}
	if err := database.UnapproveUser(m.ChatID(), userID); err != nil {
		_, e := m.Reply("⚠️ Failed to unapprove user.")
		if e != nil {
			return e
		}
		return err
	}
	_, err := m.Reply(
		fmt.Sprintf("🚫 User <code>%d</code> has been unapproved.", userID),
		&telegram.SendOptions{ParseMode: "HTML"},
	)
	return err
}

func approvedCmd(m *telegram.NewMessage) error {
	userID := utils.ExtractUser(m)
	if userID == 0 {
		return telegram.ErrEndGroup
	}
	approved, err := database.IsUserApproved(m.ChatID(), userID)
	if err != nil {
		_, e := m.Reply("⚠️ Failed to check approval status.")
		if e != nil {
			return e
		}
		return err
	}
	status := "not approved"
	if approved {
		status = "approved ✅"
	}
	_, err = m.Reply(
		fmt.Sprintf("👤 User <code>%d</code> is %s.", userID, status),
		&telegram.SendOptions{ParseMode: "HTML"},
	)
	return err
}
