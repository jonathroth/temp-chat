package state

import (
	"database/sql"
	"fmt"
	"time"

	// PostgreSQL package driver, used by sql.Open()
	_ "github.com/lib/pq"
)

const (
	createServersTable = `CREATE TABLE IF NOT EXISTS servers (
		server_id					bigint		NOT NULL	PRIMARY KEY,
		command_channel_id 			bigint		DEFAULT 0,
		temp_channel_category_id	bigint		NOT NULL,
		custom_command				varchar(32)	DEFAULT '',
		command_prefix				char(1)		DEFAULT '!',
		last_modified_timestamp		timestamp	NOT NULL,
		insertion_timestamp			timestamp	NOT NULL
	);`
	getServers             = `SELECT server_id, command_channel_id, temp_channel_category_id, custom_command, command_prefix FROM servers;`
	addServer              = `INSERT INTO servers (server_id, temp_channel_category_id, last_modified_timestamp, insertion_timestamp) VALUES ($1, $2, $3, $4);`
	getServer              = `SELECT server_id, command_channel_id, temp_channel_category_id, custom_command, command_prefix FROM servers WHERE server_id = $1;`
	updateCategoryID       = `UPDATE servers SET (temp_channel_category_id, last_modified_timestamp) = ($2, $3) WHERE server_id = $1;`
	updateCustomCommand    = `UPDATE servers SET (custom_command, last_modified_timestamp) = ($2, $3) WHERE server_id = $1;`
	updateCommandChannelID = `UPDATE servers SET (command_channel_id, last_modified_timestamp) = ($2, $3) WHERE server_id = $1;`
	updateCommandPrefix    = `UPDATE servers SET (command_prefix, last_modified_timestamp) = ($2, $3) WHERE server_id = $1;`
)

/*
// SyncServerData synchronizes read/writes to the server data.
type SyncServerData struct {
	data  *ServerData
	mutex sync.RWMutex
}

// NewSyncServerData initializes a new instance of SyncServerData.
func NewSyncServerData(data *ServerData) *SyncServerData {
	return &SyncServerData{data: data}
}

// CommandChannelID synchronizes access to ServerData.CommandChannelID
func (d *SyncServerData) CommandChannelID() uint64 {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.data.CommandChannelID
}

// SetCommandChannelID synchronizes access to ServerData.CommandChannelID
func (d *SyncServerData) SetCommandChannelID(value uint64) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.data.CommandChannelID = value
}

// TempChannelCategoryID synchronizes access to ServerData.TempChannelCategoryID
func (d *SyncServerData) TempChannelCategoryID() uint64 {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.data.TempChannelCategoryID
}

// SetTempChannelCategoryID synchronizes access to ServerData.TempChannelCategoryID
func (d *SyncServerData) SetTempChannelCategoryID(value uint64) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.data.TempChannelCategoryID = value
}

// CustomCommand synchronizes access to ServerData.CustomCommand
func (d *SyncServerData) CustomCommand() string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.data.CustomCommand
}

// SetCustomCommand synchronizes access to ServerData.CustomCommand
func (d *SyncServerData) SetCustomCommand(value string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.data.CustomCommand = value
}

// CommandPrefix synchronizes access to ServerData.CommandPrefix
func (d *SyncServerData) CommandPrefix() string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.data.CommandPrefix
}

// SetCommandPrefix synchronizes access to ServerData.CommandPrefix
func (d *SyncServerData) SetCommandPrefix(value string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.data.CommandPrefix = value
}*/

// PostgresServerData wraps server-specific
type PostgresServerData struct {
	serverID              uint64
	commandChannelID      uint64
	tempChannelCategoryID uint64
	customCommand         string
	commandPrefix         string
}

// PostgresServersProvider is a ServerProvider implementation over PostgreSQL.
type PostgresServersProvider struct {
	address string
	db      *sql.DB
}

type sqlScanner interface {
	Scan(...interface{}) error
}

// NewPostgresServersProvider initializes a new instance of PostgresServersProvider
func NewPostgresServersProvider(address string) (*PostgresServersProvider, error) {
	db, err := sql.Open("postgres", address)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(createServersTable)
	if err != nil {
		return nil, err
	}

	return &PostgresServersProvider{address: address, db: db}, nil
}

// Servers returns the list of all servers managed by the bot.
func (p *PostgresServersProvider) Servers() (ServersData, error) {
	result := ServersData{}

	rows, err := p.db.Query(getServers)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		serverData, err := p.initializeServer(rows)
		if err != nil {
			return nil, err
		}

		result[serverData.ServerID()] = serverData
	}

	return result, nil
}

func (p *PostgresServersProvider) initializeServer(scanner sqlScanner) (ServerData, error) {
	serverData := &PostgresServerData{}
	err := scanner.Scan(&serverData.serverID, &serverData.commandChannelID, &serverData.tempChannelCategoryID, &serverData.customCommand, &serverData.commandPrefix)
	if err != nil {
		return nil, err
	}

	return serverData, nil
}

// AddServer adds a new server to the store.
func (p *PostgresServersProvider) AddServer(serverID DiscordID, tempChannelCategoryID DiscordID) (ServerData, error) {
	currentTime := time.Now().UTC()

	_, err := p.db.Exec(addServer, serverID, tempChannelCategoryID, currentTime, currentTime)
	if err != nil {
		return nil, err
	}

	return p.server(serverID)
}

func (p *PostgresServersProvider) server(serverID DiscordID) (ServerData, error) {
	return p.initializeServer(p.db.QueryRow(getServer, serverID))
}

// UpdateCategoryID updates the temp channel category ID for a server.
func (s *PostgresServerStore) UpdateCategoryID(serverID uint64, newCategoryID uint64) error {
	return assertOneChange(s.db.Exec(updateCategoryID, serverID, newCategoryID, time.Now().UTC()))
}

// UpdateCustomCommand updates the custom command for a server.
func (s *PostgresServerStore) UpdateCustomCommand(serverID uint64, customCommand string) error {
	return assertOneChange(s.db.Exec(updateCustomCommand, serverID, customCommand, time.Now().UTC()))

}

// UpdateCommandChannelID updates the custom command channel ID for a server.
func (s *PostgresServerStore) UpdateCommandChannelID(serverID uint64, commandChannelID uint64) error {
	return assertOneChange(s.db.Exec(updateCommandChannelID, serverID, commandChannelID, time.Now().UTC()))
}

// UpdateCommandPrefix updates the command prefix for a server.
func (s *PostgresServerStore) UpdateCommandPrefix(serverID uint64, newPrefix string) error {
	return assertOneChange(s.db.Exec(updateCommandPrefix, serverID, newPrefix, time.Now().UTC()))
}

func assertOneChange(sqlResult sql.Result, err error) error {
	if err != nil {
		return err
	}

	rowsAffected, err := sqlResult.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected != 1 {
		return fmt.Errorf("Expected a single row update, got %v", rowsAffected)
	}

	return nil
}
