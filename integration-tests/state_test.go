package integration_test

import (
	"errors"

	"github.com/jonathroth/temp-chat/consts"
	"github.com/jonathroth/temp-chat/state"
)

type MemoryDataProvider struct {
	database state.ServersData
}

func NewMemoryDataProvider() *MemoryDataProvider {
	return &MemoryDataProvider{database: state.ServersData{}}
}

// Servers returns the list of all servers managed by the bot.
func (p *MemoryDataProvider) Servers() (state.ServersData, error) {
	return p.database, nil
}

// AddServer adds a new server to the store.
func (p *MemoryDataProvider) AddServer(serverID state.DiscordID, tempChannelCategoryID state.DiscordID) (state.ServerData, error) {
	_, alreadyInDatabase := p.database[serverID]
	if alreadyInDatabase {
		return nil, errors.New("ID already in database, can't add")
	}

	data := NewMemoryServerData(serverID, tempChannelCategoryID)
	p.database[serverID] = data
	return data, nil
}

type MemoryServerData struct {
	serverID              state.DiscordID
	tempChannelCategoryID state.DiscordID
	commandPrefix         string
	commandChannelID      state.DiscordID
	customCommand         string
}

func NewMemoryServerData(serverID, categoryID state.DiscordID) *MemoryServerData {
	return &MemoryServerData{
		serverID:              serverID,
		tempChannelCategoryID: categoryID,
		commandPrefix:         consts.DefaultCommandPrefix,
		commandChannelID:      state.DiscordIDNone,
		customCommand:         "",
	}
}

// ServerID returns the ID of the server whose data is saved in this object.
func (d *MemoryServerData) ServerID() state.DiscordID {
	return d.serverID
}

// TempChannelCategoryID is the category Discord ID of the category to create temporary chat channels in.
func (d *MemoryServerData) TempChannelCategoryID() state.DiscordID {
	return d.tempChannelCategoryID
}

// SetTempChannelCategoryID sets a new channel category.
func (d *MemoryServerData) SetTempChannelCategoryID(value state.DiscordID) error {
	d.tempChannelCategoryID = value
	return nil
}

// CommandPrefix returns the server's specific command prefix.
func (d *MemoryServerData) CommandPrefix() string {
	return d.commandPrefix
}

// SetCustomCommandPrefix changes the command prefix to the a custom prefix.
func (d *MemoryServerData) SetCustomCommandPrefix(value string) error {
	d.commandPrefix = value
	return nil
}

// ResetCommandPrefix resets the prefix to the default value.
func (d *MemoryServerData) ResetCommandPrefix() error {
	d.commandPrefix = consts.DefaultCommandPrefix
	return nil
}

// HasDifferentPrefix returns whether the prefix was changed or not.
func (d *MemoryServerData) HasDifferentPrefix() bool {
	return d.commandPrefix != consts.DefaultCommandPrefix
}

// CommandChannelID is the ID of the channel the bot will exclusively receive commands on.
func (d *MemoryServerData) CommandChannelID() state.DiscordID {
	return d.commandChannelID
}

// SetCommandChannelID sets a specific command channel.
func (d *MemoryServerData) SetCommandChannelID(value state.DiscordID) error {
	d.commandChannelID = value
	return nil
}

// ClearCommandChannelID removes the specific command channel.
func (d *MemoryServerData) ClearCommandChannelID() error {
	d.commandChannelID = state.DiscordIDNone
	return nil
}

// HasCommandChannelID returns whether the specific command channel is set.
func (d *MemoryServerData) HasCommandChannelID() bool {
	return d.commandChannelID != state.DiscordIDNone
}

// CustomCommand is a replacement name for the make-temp-channel command name.
func (d *MemoryServerData) CustomCommand() string {
	return d.customCommand
}

// SetCustomCommand sets the replacement name for the make-temp-channel command.
func (d *MemoryServerData) SetCustomCommand(value string) error {
	d.customCommand = value
	return nil
}

// ResetCustomCommand resets the make-temp-channel command name to default.
func (d *MemoryServerData) ResetCustomCommand() error {
	d.customCommand = ""
	return nil
}

// HasCustomCommand returns whether the make-temp-channel was assigned an alternative name.
func (d *MemoryServerData) HasCustomCommand() bool {
	return d.customCommand != ""
}
