package state

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jonathroth/temp-chat/config"

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
	serverData := NewPostgresServerData(p.db)
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

// PostgresServerData wraps server-specific
type PostgresServerData struct {
	serverID              DiscordID
	commandChannelID      DiscordID
	tempChannelCategoryID DiscordID
	customCommand         string
	commandPrefix         string
	db                    *sql.DB
}

// NewPostgresServerData initializes a new instance of PostgresServerData
func NewPostgresServerData(db *sql.DB) *PostgresServerData {
	return &PostgresServerData{db: db}
}

// ServerID returns the ID of the server whose data is saved in this object.
func (d *PostgresServerData) ServerID() DiscordID {
	return d.serverID
}

// TempChannelCategoryID is the category Discord ID of the category to create temporary chat channels in.
func (d *PostgresServerData) TempChannelCategoryID() DiscordID {
	return d.tempChannelCategoryID
}

// SetTempChannelCategoryID sets a new channel category.
func (d *PostgresServerData) SetTempChannelCategoryID(value DiscordID) error {
	d.tempChannelCategoryID = value
	return assertOneChange(d.db.Exec(updateCategoryID, d.serverID, value, time.Now().UTC()))
}

// CommandPrefix returns the server's specific command prefix.
func (d *PostgresServerData) CommandPrefix() string {
	return d.commandPrefix
}

// SetCustomCommandPrefix changes the command prefix to the a custom prefix.
func (d *PostgresServerData) SetCustomCommandPrefix(value string) error {
	d.commandPrefix = value
	return assertOneChange(d.db.Exec(updateCommandPrefix, d.serverID, value, time.Now().UTC()))
}

// ResetCommandPrefix resets the prefix to the default value.
func (d *PostgresServerData) ResetCommandPrefix() error {
	return d.SetCustomCommandPrefix(config.DefaultCommandPrefix)
}

// HasDifferentPrefix returns whether the prefix was changed or not.
func (d *PostgresServerData) HasDifferentPrefix() bool {
	return d.commandPrefix != config.DefaultCommandPrefix
}

// CommandChannelID is the ID of the channel the bot will exclusively receive commands on.
func (d *PostgresServerData) CommandChannelID() DiscordID {
	return d.commandChannelID
}

// SetCommandChannelID sets a specific command channel.
func (d *PostgresServerData) SetCommandChannelID(value DiscordID) error {
	d.commandChannelID = value
	return assertOneChange(d.db.Exec(updateCommandChannelID, d.serverID, value, time.Now().UTC()))
}

// ClearCommandChannelID removes the specific command channel.
func (d *PostgresServerData) ClearCommandChannelID() error {
	return d.SetCommandChannelID(DiscordIDNone)
}

// HasCommandChannelID returns whether the specific command channel is set.
func (d *PostgresServerData) HasCommandChannelID() bool {
	return d.commandChannelID != DiscordIDNone
}

// CustomCommand is a replacement name for the make-temp-channel command name.
func (d *PostgresServerData) CustomCommand() string {
	return d.customCommand
}

// SetCustomCommand sets the replacement name for the make-temp-channel command.
func (d *PostgresServerData) SetCustomCommand(value string) error {
	d.customCommand = value
	return assertOneChange(d.db.Exec(updateCustomCommand, d.serverID, value, time.Now().UTC()))
}

// ResetCustomCommand resets the make-temp-channel command name to default.
func (d *PostgresServerData) ResetCustomCommand() error {
	return d.SetCustomCommand("")
}

// HasCustomCommand returns whether the make-temp-channel was assigned an alternative name.
func (d *PostgresServerData) HasCustomCommand() bool {
	return d.CustomCommand() != ""
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
