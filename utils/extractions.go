package utils

import (
	"strconv"
	"strings"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func fail(m *tg.NewMessage, message ...string) {
	msg := "I don't know who you're talking about, you're going to need to specify a user...!"
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}
	m.Reply(msg)
}

func extractUserIDFromEntity(m *tg.NewMessage, ent tg.MessageEntity) int64 {
	switch e := ent.(type) {

	case *tg.MessageEntityMentionName:
		return e.UserID

	case *tg.MessageEntityMention:
		username := strings.TrimPrefix(
			m.Text()[e.Offset:e.Offset+e.Length],
			"@",
		)

		peer, err := m.Client.ResolvePeer(username)
		if err != nil {
			fail(m, "Looks like I don't have control over that user.")
			return 0
		}

		user, ok := peer.(*tg.InputPeerUser)
		if !ok {
			fail(m, "This can only be used on users.")
			return 0
		}

		return user.UserID
	}

	return 0
}

func extractUserIDFallback(m *tg.NewMessage, text string) int64 {
	text = strings.TrimSpace(text)

	if id, err := strconv.ParseInt(text, 10, 64); err == nil {
		return id
	}

	peer, err := m.Client.ResolvePeer(text)
	if err != nil {
		fail(m)
		return 0
	}

	user, ok := peer.(*tg.InputPeerUser)
	if !ok {
		fail(m, "This can only be used on users.")
		return 0
	}

	return user.UserID
}

func ExtractUserAndReason(m *tg.NewMessage) (int64, string) {
	if m == nil || m.Message == nil {
		return 0, ""
	}

	text := m.Text()

	// ✅ Reply case
	if m.IsReply() {
		reply, err := m.GetReplyMessage()
		if err != nil {
			fail(m)
			return 0, ""
		}

		if _, ok := reply.Message.FromID.(*tg.PeerUser); !ok {
			fail(m, "This can only be used on users.")
			return 0, ""
		}

		return reply.SenderID(), strings.TrimSpace(m.Args())
	}

	cmd := strings.SplitN(text, " ", 2)[0]
	cmdEnd := len(cmd)

	hasEntity := false

	for _, ent := range m.Message.Entities {
		// ✅ CASE 1
		switch e := ent.(type) {

		case *tg.MessageEntityMention:
			hasEntity = true

			if e.Offset != int32(cmdEnd+1) {
				fail(m, "User mention must come right after the command.")
				return 0, ""
			}

			userID := extractUserIDFromEntity(m, e)
			if userID == 0 {
				return 0, ""
			}

			reason := strings.TrimSpace(
				text[e.Offset+e.Length:],
			)

			return userID, reason

		// ✅ CASE 2
		case *tg.MessageEntityMentionName:
			hasEntity = true

			if e.Offset != int32(cmdEnd+1) {
				fail(m, "User mention must come right after the command.")
				return 0, ""
			}

			reason := strings.TrimSpace(
				text[e.Offset+e.Length:],
			)

			return e.UserID, reason
		}
	}

	// ✅ Fallback
	if !hasEntity {
		args := strings.Fields(m.Args())
		if len(args) >= 1 {
			userID := extractUserIDFallback(m, args[0])
			if userID == 0 {
				return 0, ""
			}

			if len(args) == 1 {
				return userID, ""
			}

			return userID, strings.Join(args[1:], " ")
		}
	}

	fail(m)
	return 0, ""
}

func ExtractUser(m *tg.NewMessage) int64 {
	id, _ := ExtractUserAndReason(m)
	return id
}
