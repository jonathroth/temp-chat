package state

import (
	"fmt"
	"sync"
)

// SyncServerStore wraps a server store and sync all access
type SyncServerStore struct {
	provider ServersProvider
	servers  ServersData
	mutex    sync.RWMutex
}

// NewSyncServerStore initializes a new instance of NewSyncServerStore
func NewSyncServerStore(provider ServersProvider) (*SyncServerStore, error) {
	servers, err := provider.Servers()
	if err != nil {
		return nil, err
	}

	store := &SyncServerStore{
		provider: provider,
		servers:  servers,
	}

	return store, nil
}

// Server returns the data of a specific server managed by the bot.
func (s *SyncServerStore) Server(serverID DiscordID) (ServerData, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	serverData, found := s.servers[serverID]
	if !found {
		return nil, false
	}

	return serverData, true
}

// AddServer adds a new server to the store.
func (s *SyncServerStore) AddServer(serverID DiscordID, tempChannelCategoryID DiscordID) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	_, alreadyExists := s.servers[serverID]
	if alreadyExists {
		return fmt.Errorf("Server already exists")
	}

	serverData, err := s.provider.AddServer(serverID, tempChannelCategoryID)
	if err != nil {
		return err
	}

	s.servers[serverID] = serverData
	return nil
}

// SyncServerData synchronizes read/writes to the server data.
type SyncServerData struct {
	data  ServerData
	mutex sync.RWMutex
}

// NewSyncServerData initializes a new instance of SyncServerData.
func NewSyncServerData(data ServerData) *SyncServerData {
	return &SyncServerData{data: data}
}

// ServerID returns the ID of the server whose data is saved in this object.
func (d *SyncServerData) ServerID() DiscordID {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.data.ServerID()
}

// TempChannelCategoryID is the category Discord ID of the category to create temporary chat channels in.
func (d *SyncServerData) TempChannelCategoryID() DiscordID {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.data.TempChannelCategoryID()
}

// SetTempChannelCategoryID sets a new channel category.
func (d *SyncServerData) SetTempChannelCategoryID(value DiscordID) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.data.SetTempChannelCategoryID(value)
}

// CommandPrefix returns the server's specific command prefix.
func (d *SyncServerData) CommandPrefix() string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.data.CommandPrefix()
}

// SetCustomCommandPrefix changes the command prefix to the a custom prefix.
func (d *SyncServerData) SetCustomCommandPrefix(value string) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.data.SetCustomCommandPrefix(value)
}

// ResetCommandPrefix resets the prefix to the default value.
func (d *SyncServerData) ResetCommandPrefix() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.data.ResetCommandPrefix()
}

// HasDifferentPrefix returns whether the prefix was changed or not.
func (d *SyncServerData) HasDifferentPrefix() bool {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.data.HasDifferentPrefix()
}

// CommandChannelID is the ID of the channel the bot will exclusively receive commands on.
func (d *SyncServerData) CommandChannelID() DiscordID {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.data.CommandChannelID()
}

// SetCommandChannelID sets a specific command channel.
func (d *SyncServerData) SetCommandChannelID(value DiscordID) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.data.SetCommandChannelID(value)
}

// ClearCommandChannelID removes the specific command channel.
func (d *SyncServerData) ClearCommandChannelID() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.data.ClearCommandChannelID()
}

// HasCommandChannelID returns whether the specific command channel is set.
func (d *SyncServerData) HasCommandChannelID() bool {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.data.HasCommandChannelID()
}

// CustomCommand is a replacement name for the make-temp-channel command name.
func (d *SyncServerData) CustomCommand() string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.data.CustomCommand()
}

// SetCustomCommand sets the replacement name for the make-temp-channel command.
func (d *SyncServerData) SetCustomCommand(value string) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.data.SetCustomCommand(value)
}

// ResetCustomCommand resets the make-temp-channel command name to default.
func (d *SyncServerData) ResetCustomCommand() error {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.data.ResetCustomCommand()
}

// HasCustomCommand returns whether the make-temp-channel was assigned an alternative name.
func (d *SyncServerData) HasCustomCommand() bool {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.data.HasCustomCommand()
}
