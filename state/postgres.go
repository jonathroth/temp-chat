package state

import (
	"database/sql"
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
	updateCustomCommand    = `UPDATE servers SET (custom_command, last_modified_timestamp) = ($2, $3) WHERE server_id = $1;`
	updateCommandChannelID = `UPDATE servers SET (command_channel_id, last_modified_timestamp) = ($2, $3) WHERE server_id = $1;`
	updateCommandPrefix    = `UPDATE servers SET (command_prefix, last_modified_timestamp) = ($2, $3) WHERE server_id = $1;`
)

// PostgresServerStore manages a server store over PostgreSQL.
type PostgresServerStore struct {
	address string
	db      *sql.DB
}

// NewPostgresServerStore initializes a new instance of PostgresServerStore
func NewPostgresServerStore(address string) *PostgresServerStore {
	return &PostgresServerStore{address: address}
}

// Connect connects to database at the given address
func (s *PostgresServerStore) Connect() error {
	var err error
	s.db, err = sql.Open("postgres", s.address)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(createServersTable)
	if err != nil {
		return err
	}

	return nil
}

// Servers returns the list of all servers managed by the bot.
func (s *PostgresServerStore) Servers() (ServersData, error) {

	result := ServersData{}

	rows, err := s.db.Query(getServers)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		serverData := &ServerData{}
		err := rows.Scan(&serverData.ServerID, &serverData.CommandChannelID, &serverData.TempChannelCategoryID, &serverData.CustomCommand, &serverData.CommandPrefix)
		if err != nil {
			return nil, err
		}

		result[serverData.ServerID] = serverData
	}

	return result, nil
}

// AddServer adds a new server to the store.
func (s *PostgresServerStore) AddServer(serverID uint64, tempChannelCategoryID uint64) error {
	currentTime := time.Now().UTC()
	_, err := s.db.Exec(addServer, serverID, tempChannelCategoryID, currentTime, currentTime)
	return err
}

// UpdateCustomCommand updates the custom command for a server.
func (s *PostgresServerStore) UpdateCustomCommand(serverID uint64, customCommand string) error {
	_, err := s.db.Exec(updateCustomCommand, serverID, customCommand, time.Now().UTC())
	return err
}

// UpdateCommandChannelID updates the custom command channel ID for a server.
func (s *PostgresServerStore) UpdateCommandChannelID(serverID uint64, commandChannelID uint64) error {
	_, err := s.db.Exec(updateCommandChannelID, serverID, commandChannelID, time.Now().UTC())
	return err
}

// UpdateCommandPrefix updates the command prefix for a server.
func (s *PostgresServerStore) UpdateCommandPrefix(serverID uint64, newPrefix string) error {
	_, err := s.db.Exec(updateCommandChannelID, serverID, newPrefix, time.Now().UTC())
	return err
}
