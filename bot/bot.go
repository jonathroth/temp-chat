package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/jonathroth/temp-chat/state"
)

// TempChannelBot contains all the handlers to discord events for the bot to operate.
type TempChannelBot struct {
	store     state.ServerStore
	botUserID state.DiscordID

	tempChannels *TempChannelList

	commands map[string]*Command
}

// NewTempChannelBot initializes a new instance of TempChannelBot.
func NewTempChannelBot(session *discordgo.Session, store state.ServerStore) (*TempChannelBot, error) {
	user, err := session.User("@me")
	if err != nil {
		return nil, err
	}

	userID, err := state.ParseDiscordID(user.ID)
	if err != nil {
		return nil, fmt.Errorf("Failed parsing self user ID: %v", err)
	}

	bot := &TempChannelBot{
		store:        store,
		botUserID:    userID,
		tempChannels: NewTempChannelList(session),
	}
	bot.commands = bot.initCommands()

	return bot, nil
}

// CleanChannels deletes all the temp channels created by the bot.
func (b *TempChannelBot) CleanChannels() {
	for _, tempChannel := range b.tempChannels.tempChannelIDToTempChannel {
		b.tempChannels.DeleteTempChannel(tempChannel)
	}
}
