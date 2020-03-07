package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jonathroth/temp-chat/state"
)

const (
	makeChannelCommand = "mkch"
	defaultPrefix      = "!"
)

func (b *TempChannelBot) initCommands() map[string]*Command {
	return map[string]*Command{
		makeChannelCommand: &Command{SetupRequired: true, AdminOnly: false},
		"set-mkch":         &Command{SetupRequired: true, AdminOnly: true},
		"set-prefix":       &Command{SetupRequired: true, AdminOnly: true, Handler: b.setPrefixHandler},
		"set-command-ch":   &Command{SetupRequired: true, AdminOnly: true},
		"setup":            &Command{SetupRequired: false, AdminOnly: true, Handler: b.setupHandler},
		"help":             &Command{SetupRequired: false, AdminOnly: false, Handler: helpHandler},
	}
}

// CommandHandler is a handler func called when a command is successfully parsed.
// An error returned by the handler will cause the bot to exit with a log.Fatal.
type CommandHandler func(s *discordgo.Session, m *discordgo.MessageCreate, args []string, serverData *state.ServerData) error

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

	command, found := b.commands[commandParts[0]]
	if !found {
		b.replyToSenderAndLog(s, m.ChannelID, "Unknown command %q", commandParts[0])
		return
	}

	if command.SetupRequired && !serverIsSetup {
		b.replyToSenderAndLog(s, m.ChannelID, "The bot hasn't been set up yet, please use %vsetup first", prefix)
		return
	}

	// TODO: handle command.AdminOnly

	err = command.Handler(s, m, commandParts, serverData)
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

func helpHandler(s *discordgo.Session, m *discordgo.MessageCreate, args []string, serverData *state.ServerData) error {
	_, err := s.ChannelMessageSend(m.ChannelID, "```"+`less
[TempChat]
TempChat is a bot that creates temporary text channels for Discord voice chats.

The bot will give permission to any new user that joins the voice chat, and revoke the permission to any user that leaves it.
All channels are created under a specific category, the category must give the bot account the [Manage Channel] permission, and must deny the [Read Text Channels & See Voice Channels] from @everyone

#Commands:
!help - Displays this menu

[Before Setup]
!setup [category-id] - Configured the category ID the bot should create the temporary channels in

[After Setup]
!setup [category-id] - Changes the category ID to new category
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

func (b *TempChannelBot) setupHandler(s *discordgo.Session, m *discordgo.MessageCreate, args []string, serverData *state.ServerData) error {
	if len(args) != 2 {
		b.replyToSenderAndLog(s, m.ChannelID, "Category ID is missing")
		return nil
	}

	categoryID, err := parseID(args[1])
	if err != nil {
		b.replyToSenderAndLog(s, m.ChannelID, `Invalid category ID, please right click the category and click "Copy ID"`)
		log.Printf("Invalid category ID %q: %v", args[1], err)
		return nil
	}

	channel, err := s.State.Channel(args[1])
	if !existsInState(err) || channel.GuildID != m.GuildID {
		b.replyToSenderAndLog(s, m.ChannelID, `This category doesn't exist, please right click the category and click "Copy ID"`)
		return nil
	}

	serverID, _ := parseID(m.GuildID)
	_, alreadySetup := b.servers[serverID]
	if alreadySetup {
		serverData.TempChannelCategoryID = categoryID
		err := b.store.UpdateCategoryID(serverID, categoryID)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "An internal error has occurred")
			return fmt.Errorf("UpdateCategoryID failed: %v", err)
		}

		b.replyToSenderAndLog(s, m.ChannelID, `Category ID updated successfully`)
		return nil
	} else {
		newServerData, err := b.store.AddServer(serverID, categoryID)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "An internal error has occurred")
			return fmt.Errorf("AddServer failed: %v", err)
		}

		b.servers[serverID] = newServerData
		b.replyToSenderAndLog(s, m.ChannelID, `Server was setup successfully, you may use %v%v`, defaultPrefix, makeChannelCommand)
		return nil
	}
}

func (b *TempChannelBot) setPrefixHandler(s *discordgo.Session, m *discordgo.MessageCreate, args []string, serverData *state.ServerData) error {
	if len(args) > 2 {
		b.replyToSenderAndLog(s, m.ChannelID, "Too many arguments, please check !help to see how to use the command")
		return nil
	}

	if len(args) == 1 {
		if serverData.CommandPrefix == "" || serverData.CommandPrefix == defaultPrefix {
			b.replyToSenderAndLog(s, m.ChannelID, "The prefix is already set to %v, please check !help to see how to use the command", defaultPrefix)
			return nil
		}
		serverData.CommandPrefix = defaultPrefix
		err := b.store.UpdateCommandPrefix(serverData.ServerID, defaultPrefix)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "An internal error has occurred")
			return fmt.Errorf("UpdateCommandPrefix failed: %v", err)
		}
	} else if len(args) == 2 {
		newPrefix := args[1]
		if len(newPrefix) != 1 {
			b.replyToSenderAndLog(s, m.ChannelID, "The prefix must be exactly 1 character")
			return nil
		}

		serverData.CommandPrefix = newPrefix
		err := b.store.UpdateCommandPrefix(serverData.ServerID, newPrefix)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "An internal error has occurred")
			return fmt.Errorf("UpdateCommandPrefix failed: %v", err)
		}
	}
	b.replyToSenderAndLog(s, m.ChannelID, "Prefix changed successfully")
	return nil
}

func existsInState(err error) bool {
	return err != discordgo.ErrStateNotFound
}
