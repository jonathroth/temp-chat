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
var Commands = map[string]*Command{
	makeChannelCommand: &Command{SetupRequired: true, AdminOnly: false},
	"set-mkch":         &Command{SetupRequired: true, AdminOnly: true},
	"set-prefix":       &Command{SetupRequired: true, AdminOnly: true},
	"set-command-ch":   &Command{SetupRequired: true, AdminOnly: true},
	"setup":            &Command{SetupRequired: false, AdminOnly: true},
	"help":             &Command{SetupRequired: false, AdminOnly: false, Handler: helpHandler},
}

// CommandHandler is a handler func called when a command is successfully parsed.
// An error returned by the handler will cause the bot to exit with a log.Fatal.
type CommandHandler func(s *discordgo.Session, m *discordgo.MessageCreate, args []string) error

// Command defines the logic and conditions required for a command to run.
type Command struct {
	// SetupRequired tells whether the command can be run before the bot was configured for the server.
	SetupRequired bool
	AdminOnly     bool
	Handler       CommandHandler
}

// MessageCreate is called whenever a message arrives in a server the bot is in.
func (b *TempChannelBot) MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == b.botUserID || m.Author.Bot {
		return
	}

	serverID, err := parseID(m.GuildID)
	if err != nil {
		log.Fatalf("Failed to parse discord ID: %v", err)
	}

	serverData, serverIsSetup := b.servers[serverID]

	prefix := defaultPrefix
	if serverIsSetup && serverData.CommandPrefix != "" {
		prefix = serverData.CommandPrefix
	}

	if !strings.HasPrefix(m.Content, prefix) {
		// Not a command, ignore.
		return
	}

	commandText := strings.TrimPrefix(m.Content, prefix)
	commandParts := strings.Split(commandText, " ")

	if serverIsSetup && serverData.CustomCommand != "" && commandParts[0] == serverData.CustomCommand {
		commandParts[0] = makeChannelCommand
	}

	command, found := Commands[commandParts[0]]
	if !found {
		b.replyToSenderAndLog(s, m.ChannelID, "Unknown command %q", commandParts[0])
		return
	}

	if command.SetupRequired && !serverIsSetup {
		b.replyToSenderAndLog(s, m.ChannelID, "The bot hasn't been set up yet, please use %vsetup first", prefix)
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

func helpHandler(s *discordgo.Session, m *discordgo.MessageCreate, args []string) error {
	_, err := s.ChannelMessageSend(m.ChannelID, "```"+`less
[TempChat]
TempChat is a bot that creates temporary text channels for Discord voice chats.

The bot will give permission to any new user that joins the voice chat, and revoke the permission to any user that leaves it.
All channels are created under a specific category, the category must give the bot account the [Manage Channel] permission, and must deny the [Read Text Channels & See Voice Channels] from @everyone

#Commands:
[Before Setup]
!help - Displays this menu
!setup [category-id] - Sets up the bot

[After Setup]
!mkch - Creates a temporary voice channel for the users in your voice chat
!set-prefix [new-prefix] - Changes the command prefix
[?]set-prefix - Resets the command prefix to !
!set-mkch [new-name] - Changes the !mkch command to the desired command name
!set-mkch - Resets the command name to !mkch
!set-command-ch [channel-name] - Sets a specific channel for the bot to read commands from, the bot will ignore all other channels.
!set-command-ch - Removes the specified command channel`+"```")
	if err != nil {
		return err
	}

	return nil
}
