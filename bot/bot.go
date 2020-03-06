package bot

import "github.com/jonathroth/temp-chat/state"

// TempChannelBot contains all the handlers to discord events for the bot to operate.
type TempChannelBot struct {
	store   state.ServerStore
	servers state.ServersData
}

// NewTempChannelBot initializes a new instance of TempChannelBot.
func NewTempChannelBot(store state.ServerStore) (*TempChannelBot, error) {
	servers, err := store.Servers()
	if err != nil {
		return nil, err
	}

	bot := &TempChannelBot{
		store:   store,
		servers: servers,
	}
	return bot, nil
}
