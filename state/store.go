package state

// ServerData is a single server monitored by the bot.
type ServerData struct {
	ServerID              uint64
	CommandChannelID      uint64
	TempChannelCategoryID uint64
	CustomCommand         string
}

// ServerStore manages the state of servers managed by the bot.
type ServerStore interface {
	// Servers returns the list of all servers managed by the bot.
	Servers() ([]*ServerData, error)

	// AddServer adds a new server to the store.
	AddServer(serverID uint64, tempChannelCategoryID uint64) error

	// UpdateCustomCommand updates the custom command for a server.
	UpdateCustomCommand(serverID uint64, customCommand string) error

	// UpdateCommandChannelID updates the custom command channel ID for a server.
	UpdateCommandChannelID(serverID uint64, commandChannelID uint64) error
}
