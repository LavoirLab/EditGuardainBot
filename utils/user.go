package utils

import (
	"fmt"
	"html"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func MentionHTML(u *tg.UserObj) string {
	if u == nil {
		return "Unknown"
	}

	fullName := u.FirstName
	if u.LastName != "" {
		fullName += " " + u.LastName
	}

	if fullName == "" {
		fullName = "User"
	}
	fullName = html.EscapeString(fullName)

	return fmt.Sprintf("<a href=\"tg://user?id=%d\">%s</a>", u.ID, fullName)
}
