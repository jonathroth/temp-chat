package state

import (
	"errors"
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
func (s *SyncServerStore) Server(serverID DiscordID) (ServerData, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	serverData, found := s.servers[serverID]
	if !found {
		return nil, errors.New("Unknown server ID")
	}

	return serverData, nil
}

// AddServer adds a new server to the store.
func (s *SyncServerStore) AddServer(serverID DiscordID, tempChannelCategoryID DiscordID) (ServerData, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	newServerData, err := s.provider.AddServer(serverID, tempChannelCategoryID)
	if err != nil {
		return nil, err
	}

	s.servers[serverID] = newServerData
	return newServerData, nil
}

type SyncServerData struct {
	ServerData
	mutex sync.RWMutex
}
