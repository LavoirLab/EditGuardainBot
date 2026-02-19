package modules

import (
	"fmt"
	"html"

	"github.com/amarnathcjd/gogram/telegram"

	"main/utils"
)

func deleteEditedMessage(m *telegram.NewMessage) error {
	if !IsSupergroup(m) || m.Message.EditHide {
		return nil
	}
	if ShouldIgnoreGroupAnonymous(m) {
		return nil
	}
	if isadmin, err := utils.IsChatAdmin(m.Client, m.ChatID(), m.Sender.ID); err != nil {
		L(m, "Modules -> edit -> utils.IsChatAdmin()", err)
		return nil
	} else if isadmin {
		return nil
	}

	if _, err := m.Delete(); err != nil && handleNeedPerm(err, m) {
		return telegram.EndGroup
	} else if err != nil {
		return L(m, "Modules -> edit -> m.Delete()", err)
	}

	var senderTag string
	if m.Sender.Username != "" {
		senderTag = "@" + m.Sender.Username
	} else {
		senderTag = fmt.Sprintf(`<a href="tg://user?id=%d">%s</a>`, m.Sender.ID, html.EscapeString(m.Sender.FirstName))
	}

	reason := fmt.Sprintf(`<b>🚫 %s edited a message.</b> Editing messages is prohibited in this chat to maintain conversation integrity.`, senderTag)
	p := fmt.Sprintf(`<b>📷 %s edited a photo caption.</b> Image edits are blocked to preserve context.`, senderTag)
	v := fmt.Sprintf(`<b>🎥 %s edited a video caption.</b> Video edits aren't allowed to retain originality.`, senderTag)
	d := fmt.Sprintf(`<b>📄 %s edited a document caption.</b> Please avoid modifying documents.`, senderTag)
	a := fmt.Sprintf(`<b>🎵 %s edited an audio caption.</b> Audio files must remain unaltered.`, senderTag)

	switch {
	case m.Text() != "" && m.IsMedia():
		reason = fmt.Sprintf(`<b>✍️ %s edited a media caption.</b> Caption edits affect clarity and are not permitted.`, senderTag)
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
		reason = fmt.Sprintf(`<b>🚫 %s edited text.</b> Editing text is not allowed to keep conversations clear.`, senderTag)
	case m.Photo() != nil:
		reason = fmt.Sprintf(`<b>📷 %s edited a photo.</b> Editing photos is not allowed.`, senderTag)
	case m.Video() != nil:
		reason = fmt.Sprintf(`<b>🎥 %s edited a video.</b> Editing videos is not permitted.`, senderTag)
	case m.Document() != nil:
		reason = fmt.Sprintf(`<b>📄 %s edited a document.</b> Document edits are not allowed.`, senderTag)
	case m.Audio() != nil:
		reason = fmt.Sprintf(`<b>🎵 %s edited an audio file.</b> Audio editing is not permitted.`, senderTag)
	case m.Voice() != nil:
		reason = fmt.Sprintf(`<b>🎙️ %s edited a voice message.</b> Voice messages should remain original.`, senderTag)
	case m.Animation() != nil:
		reason = fmt.Sprintf(`<b>🎞️ %s edited a GIF or animation.</b> Keep animations unchanged for context.`, senderTag)
	}

	_, err := m.Respond(reason)
	return L(m, "Modules -> edit -> respond", err)
}
