package state

import "strconv"

// DiscordID is a unique identifier used by the Discord API.
type DiscordID uint64

// DiscordIDNone is used to tell or check if an ID isn't configured.
const DiscordIDNone = DiscordID(0)

// ParseDiscordID parses a Discord REST API ID into a DiscordID.
func ParseDiscordID(id string) (DiscordID, error) {
	discordID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return DiscordIDNone, err
	}

	return DiscordID(discordID), nil
}

// RESTAPIFormat returns the Discord ID in the Discord REST API format.
func (i DiscordID) RESTAPIFormat() string {
	return strconv.FormatUint(uint64(i), 10)
}

// String returns a string form of the ID.
func (i DiscordID) String() string {
	return i.RESTAPIFormat()
}

// ServerData is a single server monitored by the bot.
type ServerData interface {
	// ServerID returns the ID of the server whose data is saved in this object.
	ServerID() DiscordID

	// TempChannelCategoryID is the category Discord ID of the category to create temporary chat channels in.
	TempChannelCategoryID() DiscordID
	// SetTempChannelCategoryID sets a new channel category.
	SetTempChannelCategoryID(value DiscordID) error

	// CommandPrefix returns the server's specific command prefix.
	CommandPrefix() string
	// SetCommandPrefix changes the command prefix to the new prefix
	SetCommandPrefix(value string) error
	// ResetCommandPrefix resets the prefix to the default value.
	ResetCommandPrefix() error

	// CommandChannelID is the ID of the channel the bot will exclusively receive commands on.
	CommandChannelID() DiscordID
	// SetCommandChannelID sets a specific command channel.
	SetCommandChannelID(value DiscordID) error
	// ClearCommandChannelID removes the specific command channel.
	ClearCommandChannelID() error
	// HasCommandChannelID returns whether the specific command channel is set.
	HasCommandChannelID() bool

	// CustomCommand is a replacement name for the make-temp-channel command name.
	CustomCommand() string
	// SetCustomCommand sets the replacement name for the make-temp-channel command.
	SetCustomCommand(value string) error
	// ResetCustomCommand resets the make-temp-channel command name to default.
	ResetCustomCommand(value string) error
	// HasCustomCommand returns whether the make-temp-channel was assigned an alternative name.
	HasCustomCommand() bool
}

// ServersData maps from the server ID to the relevant server data struct.
type ServersData map[DiscordID]ServerData

// ServerStore is used by the bot to access specific server data and add new servers
type ServerStore interface {
	// Server returns the data of a specific server managed by the bot.
	Server(serverID DiscordID) (ServerData, error)

	// AddServer adds a new server to the store.
	AddServer(serverID DiscordID, tempChannelCategoryID DiscordID) (ServerData, error)
}

// ServersProvider provides the server data to the store from the database.
type ServersProvider interface {
	// Servers returns the list of all servers managed by the bot.
	Servers() (ServersData, error)

	// AddServer adds a new server to the store.
	AddServer(serverID DiscordID, tempChannelCategoryID DiscordID) (ServerData, error)
}
