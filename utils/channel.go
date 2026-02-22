package utils

import (
	"fmt"

	"github.com/amarnathcjd/gogram/telegram"
)

func GetFullChannel(
	c *telegram.Client,
	chatId any,
) (*telegram.ChannelFull, error) {
	resolvedPeer, err := c.ResolvePeer(chatId)
	if err != nil {
		return nil, err
	}

	inPeer, ok := resolvedPeer.(*telegram.InputPeerChannel)
	if !ok {
		return nil, fmt.Errorf("chatId is not a channel")
	}

	fullChatRaw, err := c.ChannelsGetFullChannel(&telegram.InputChannelObj{
		ChannelID:  inPeer.ChannelID,
		AccessHash: inPeer.AccessHash,
	})
	if err != nil {
		return nil, err
	}

	if fullChatRaw == nil {
		return nil, fmt.Errorf("fullChatRaw is nil")
	}

	fullChat, ok := fullChatRaw.FullChat.(*telegram.ChannelFull)
	if !ok {
		return nil, fmt.Errorf("fullChatRaw.FullChat is not a ChannelFull")
	}

	return fullChat, nil
}

// GetUser fetches a Telegram user by ID without using cache.
func GetUser(client *telegram.Client, userID int64) (*telegram.UserObj, error) {
	input := &telegram.InputUserObj{
		UserID:     userID,
		AccessHash: 0, // If you're sure you don't need access hash, keep 0
	}

	users, err := client.UsersGetUsers([]telegram.InputUser{input})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("no user found with id '%d'", userID)
	}

	user, ok := users[0].(*telegram.UserObj)
	if !ok {
		return nil, fmt.Errorf(
			"expected UserObj for id '%d', but got different type",
			userID,
		)
	}

	return user, nil
}
