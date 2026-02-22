package modules

import (
	"fmt"

	"github.com/amarnathcjd/gogram/telegram"

	"main/utils"
)

func init() {
	RegisterModule(&ModuleInfo{
		Name:    "edit",
		ModName: "✏️ Edit",
		Handlers: []HandlerConfig{
			{
				Pattern: "edit",
				Handler: deleteEditedMessage,
			},
		},
	})
}

func deleteEditedMessage(m *telegram.NewMessage) error {
	if !m.IsGroup() || m.Channel == nil || m.Message.EditHide ||
		m.IsAnonymous() ||
		m.IsChannelPost() {
		return nil
	}

	isadmin, _ := utils.IsAdmin(m.Client, m.ChatID(), m.SenderID())
	if isadmin {
		return nil
	}

	if _, err := m.Delete(); err != nil {
		return err
	}

	senderTag := utils.MentionHTML(m.Sender)

	reason := fmt.Sprintf(
		`<b>🚫 %s edited a message.</b> Editing messages is prohibited in this chat to maintain conversation integrity.`,
		senderTag,
	)
	p := fmt.Sprintf(
		`<b>📷 %s edited a photo caption.</b> Image edits are blocked to preserve context.`,
		senderTag,
	)
	v := fmt.Sprintf(
		`<b>🎥 %s edited a video caption.</b> Video edits aren't allowed to retain originality.`,
		senderTag,
	)
	d := fmt.Sprintf(
		`<b>📄 %s edited a document caption.</b> Please avoid modifying documents.`,
		senderTag,
	)
	a := fmt.Sprintf(
		`<b>🎵 %s edited an audio caption.</b> Audio files must remain unaltered.`,
		senderTag,
	)

	switch {
	case m.Text() != "" && m.IsMedia():
		reason = fmt.Sprintf(
			`<b>✍️ %s edited a media caption.</b> Caption edits affect clarity and are not permitted.`,
			senderTag,
		)
		if m.Photo() != nil {
			reason = p
		} else if m.Video() != nil {
			reason = v
		} else if m.Document() != nil {
			reason = d
		} else if m.Audio() != nil {
			reason = a
		}
	case m.Text() != "" && !m.IsMedia():
		reason = fmt.Sprintf(
			`<b>🚫 %s edited text.</b> Editing text is not allowed to keep conversations clear.`,
			senderTag,
		)
	case m.Photo() != nil:
		reason = fmt.Sprintf(
			`<b>📷 %s edited a photo.</b> Editing photos is not allowed.`,
			senderTag,
		)
	case m.Video() != nil:
		reason = fmt.Sprintf(
			`<b>🎥 %s edited a video.</b> Editing videos is not permitted.`,
			senderTag,
		)
	case m.Document() != nil:
		reason = fmt.Sprintf(
			`<b>📄 %s edited a document.</b> Document edits are not allowed.`,
			senderTag,
		)
	case m.Audio() != nil:
		reason = fmt.Sprintf(
			`<b>🎵 %s edited an audio file.</b> Audio editing is not permitted.`,
			senderTag,
		)
	case m.Voice() != nil:
		reason = fmt.Sprintf(
			`<b>🎙️ %s edited a voice message.</b> Voice messages should remain original.`,
			senderTag,
		)
	case m.Animation() != nil:
		reason = fmt.Sprintf(
			`<b>🎞️ %s edited a GIF or animation.</b> Keep animations unchanged for context.`,
			senderTag,
		)
	}

	_, err := m.Reply(reason)
	return err
}
