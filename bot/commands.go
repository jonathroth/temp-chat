package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const (
	makeChannelCommand = "mkch"
	defaultPrefix      = "!"
)

// Commands maps between the command name and the command logic.
var Commands = map[string]Command{
	makeChannelCommand: Command{AdminOnly: false},
	"set-mkch":         Command{AdminOnly: true},
	"set-command-ch":   Command{AdminOnly: true},
	"setup":            Command{AdminOnly: true},
}

// CommandHandler is a handler func called when a command is successfully parsed.
// An error returned by the handler will cause the bot to exit with a log.Fatal.
type CommandHandler func(s *discordgo.Session, m *discordgo.MessageCreate, args []string) error

// Command defines the logic and conditions required for a command to run.
type Command struct {
	AdminOnly bool
	Handler   CommandHandler
}

// MessageCreate is called whenever a message arrives in a server the bot is in.
func (b *TempChannelBot) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	serverID, err := parseID(m.GuildID)
	if err != nil {
		log.Fatalf("Failed to parse discord ID: %v", err)
	}

	serverData, found := b.servers[serverID]
	if !found {
		// The server owner still hasn't called the setup command
		// Avoid logging/sending a message to not flood the logs/servers
		return
	}

	prefix := defaultPrefix
	if serverData.CommandPrefix != "" {
		prefix = serverData.CommandPrefix
	}

	if !strings.HasPrefix(m.Content, prefix) {
		// Not a command, ignore.
		return
	}

	commandText := strings.TrimPrefix(m.Content, prefix)
	commandParts := strings.Split(commandText, " ")

	if serverData.CustomCommand != "" && commandParts[0] == serverData.CustomCommand {
		commandParts[0] = makeChannelCommand
	}

	command, found := Commands[commandParts[0]]
	if !found {
		b.replyToSenderAndLog(s, m.ChannelID, "Unknown command %v", commandParts[0])
		return
	}

	// TODO: handle command.AdminOnly

	err = command.Handler(s, m, commandParts)
	if err != nil {
		log.Fatalf("Command handler %v failed: %v", commandParts[0], err)
	}
}

func (b *TempChannelBot) replyToSenderAndLog(s *discordgo.Session, channelID string, message string, args ...interface{}) {
	log.Printf(message, args...)

	_, err := s.ChannelMessageSend(channelID, fmt.Sprintf(message, args...))
	if err != nil {
		log.Fatalf("Failed sending message response: %v", err)
	}
}
