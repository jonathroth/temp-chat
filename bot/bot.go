package bot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/jonathroth/temp-chat/state"
)

// TempChannelBot contains all the handlers to discord events for the bot to operate.
type TempChannelBot struct {
	store     state.ServerStore
	servers   state.ServersData
	botUserID string

	commands map[string]*Command
}

// NewTempChannelBot initializes a new instance of TempChannelBot.
func NewTempChannelBot(s *discordgo.Session, store state.ServerStore) (*TempChannelBot, error) {
	servers, err := store.Servers()
	if err != nil {
		return nil, err
	}

	user, err := s.User("@me")
	if err != nil {
		return nil, err
	}

	bot := &TempChannelBot{
		store:     store,
		servers:   servers,
		botUserID: user.ID,
	}
	bot.commands = bot.initCommands()

	return bot, nil
}
