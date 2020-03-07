package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/jonathroth/temp-chat/config"
	"github.com/jonathroth/temp-chat/state"
)

func (b *TempChannelBot) initCommands() map[string]*Command {
	return map[string]*Command{
		config.DefaultMakeChannelCommand: &Command{SetupRequired: true, AdminOnly: false},
		"set-mkch":                       &Command{SetupRequired: true, AdminOnly: true},
		"set-prefix":                     &Command{SetupRequired: true, AdminOnly: true, Handler: b.setPrefixHandler},
		"set-command-ch":                 &Command{SetupRequired: true, AdminOnly: true, Handler: b.setCommandChannelHandler},
		"setup":                          &Command{SetupRequired: false, AdminOnly: true, Handler: b.setupHandler},
		"help":                           &Command{SetupRequired: false, AdminOnly: false, Handler: helpHandler},
	}
}

// CommandHandlerContext are the parameters passed to a command handler.
type CommandHandlerContext struct {
	Session *discordgo.Session
	Event   *discordgo.MessageCreate

	ServerID   state.DiscordID
	ServerData state.ServerData

	CommandName string
	CommandArgs []string
}

func (c *CommandHandlerContext) replyUnformatted(message string) {
	_, err := c.Session.ChannelMessageSend(c.Event.ChannelID, message)
	if err != nil {
		log.Fatalf("Failed sending message response: %v", err)
	}
}

func (c *CommandHandlerContext) reply(message string, args ...interface{}) {
	c.replyUnformatted("`" + fmt.Sprintf(message, args...) + "`")
}

func (c *CommandHandlerContext) logAndReply(message string, args ...interface{}) {
	log.Printf(message, args...)
	c.reply(message, args...)
}

func (c *CommandHandlerContext) categoryExists(categoryID string) bool {
	channel, err := c.Session.State.Channel(categoryID)
	return c.existsInState(err) && channel.GuildID == c.Event.GuildID && channel.Type == discordgo.ChannelTypeGuildCategory
}

func (c *CommandHandlerContext) textChannelExists(channelID string) bool {
	channel, err := c.Session.State.Channel(channelID)
	return c.existsInState(err) && channel.GuildID == c.Event.GuildID && channel.Type == discordgo.ChannelTypeGuildText
}

func (c *CommandHandlerContext) channelExists(channelID string) bool {
	channel, err := c.Session.State.Channel(channelID)
	return c.existsInState(err) && channel.GuildID == c.Event.GuildID
}

func (c *CommandHandlerContext) existsInState(err error) bool {
	return err != discordgo.ErrStateNotFound
}

// CommandHandler is a handler func called when a command is successfully parsed.
// An error returned by the handler will cause the bot to exit with a log.Fatal.
type CommandHandler func(*CommandHandlerContext) error

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

	serverID, err := state.ParseDiscordID(m.GuildID)
	if err != nil {
		log.Fatalf("Failed to parse discord server ID: %v", err)
	}

	serverData, serverIsSetup := b.store.Server(serverID)
	context := &CommandHandlerContext{
		Session:    s,
		Event:      m,
		ServerID:   serverID,
		ServerData: serverData,
	}

	prefix := config.DefaultCommandPrefix
	if serverIsSetup {
		prefix = serverData.CommandPrefix()
	}

	if !strings.HasPrefix(m.Content, prefix) {
		// Not a command, ignore.
		return
	}

	if serverIsSetup && serverData.HasCommandChannelID() && serverData.CommandChannelID().NotEquals(m.ChannelID) {
		if !context.channelExists(serverData.CommandChannelID().RESTAPIFormat()) {
			err := serverData.ClearCommandChannelID()
			if err != nil {
				context.reply("An internal error has occurred")
				log.Fatalf("ClearCommandChannelID failed: %v", err)
			}

			context.logAndReply("The custom command channel was deleted, and is therefore unset")
		} else {
			// Command was posted in a channel that's not the command channel
			return
		}
	}

	commandText := strings.TrimPrefix(m.Content, prefix)
	commandParts := strings.Split(commandText, " ")

	if serverIsSetup && serverData.HasCustomCommand() && commandParts[0] == serverData.CustomCommand() {
		commandParts[0] = config.DefaultMakeChannelCommand
	}

	context.CommandName = commandParts[0]
	context.CommandArgs = commandParts[1:]

	command, found := b.commands[commandParts[0]]
	if !found {
		context.reply("Unknown command %q", context.CommandName)
		return
	}

	if command.SetupRequired && !serverIsSetup {
		context.logAndReply("The bot hasn't been set up yet, please use %vsetup first", prefix)
		return
	}

	// TODO: handle command.AdminOnly

	err = command.Handler(context)
	if err != nil {
		log.Fatalf("Command handler %v failed: %v", context.CommandName, err)
	}
}

func helpHandler(context *CommandHandlerContext) error {
	context.replyUnformatted("```" + `less
[TempChat]
TempChat is a bot that creates temporary text channels for Discord voice chats.

The bot will give permission to any new user that joins the voice chat, and revoke the permission to any user that leaves it.
All channels are created under a specific category, the category must give the bot account the [Manage Channel] permission, and must deny the [Read Text Channels & See Voice Channels] from @everyone

As of now, the bot requires [Developer Mode] to be active in order to use the setup/configuration commands.

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
!set-command-ch - Removes the specified command channel` + "```")
	return nil
}

func (b *TempChannelBot) setupHandler(context *CommandHandlerContext) error {
	if len(context.CommandArgs) < 1 {
		context.reply("Missing category ID, please check %vhelp to see how to use the command", config.DefaultCommandPrefix)
		return nil
	} else if len(context.CommandArgs) > 1 {
		context.reply("Too many arguments, please check %vhelp to see how to use the command", config.DefaultCommandPrefix)
		return nil
	}

	categoryIDStr := context.CommandArgs[0]
	categoryID, err := state.ParseDiscordID(categoryIDStr)
	if err != nil {
		context.reply(`Invalid category ID, please right click the category and click "Copy ID"`)
		log.Printf("Invalid category ID %q: %v", categoryIDStr, err) // TODO: consider log level error
		return nil
	}

	if !context.channelExists(categoryIDStr) {
		context.reply(`This category doesn't exist, please right click the category and click "Copy ID"`)
		return nil
	}

	if !context.categoryExists(categoryIDStr) {
		context.reply(`The given ID isn't of a category, please right click the category and click "Copy ID"`)
		return nil
	}

	serverAlreadySetup := context.ServerData != nil
	if serverAlreadySetup {
		if context.ServerData.TempChannelCategoryID() == categoryID {
			context.reply("The server is already set up to work with the given category")
			return nil
		}

		err := context.ServerData.SetTempChannelCategoryID(categoryID)
		if err != nil {
			context.reply("An internal error has occurred")
			return fmt.Errorf("SetTempChannelCategoryID failed: %v", err)
		}

		context.reply("Category ID updated successfully")
		return nil
	}

	err = b.store.AddServer(context.ServerID, categoryID)
	if err != nil {
		context.reply("An internal error has occurred")
		return fmt.Errorf("AddServer failed: %v", err)
	}

	context.logAndReply("Server was setup successfully, you may use %v%v", config.DefaultCommandPrefix, config.DefaultMakeChannelCommand)
	return nil
}

func (b *TempChannelBot) setPrefixHandler(context *CommandHandlerContext) error {
	if len(context.CommandArgs) > 1 {
		context.reply("Too many arguments, please check %vhelp to see how to use the command", context.ServerData.CommandPrefix())
		return nil
	}

	if len(context.CommandArgs) == 0 {
		if !context.ServerData.HasDifferentPrefix() {
			context.reply("The prefix is already set to %v, please check %vhelp to see how to use the command", config.DefaultCommandPrefix, config.DefaultCommandPrefix)
			return nil
		}

		err := context.ServerData.ResetCommandPrefix()
		if err != nil {
			context.reply("An internal error has occurred")
			return fmt.Errorf("ResetCommandPrefix failed: %v", err)
		}

		context.reply("Prefix reset successfully")
	} else if len(context.CommandArgs) == 1 {
		newPrefix := context.CommandArgs[0]
		if len(newPrefix) != 1 {
			context.reply("The prefix must be exactly 1 character")
			return nil
		}

		// TODO: validate prefix is a symbol character

		err := context.ServerData.SetCustomCommandPrefix(newPrefix)
		if err != nil {
			context.reply("An internal error has occurred")
			return fmt.Errorf("SetCustomCommandPrefix failed: %v", err)
		}

		context.reply("Prefix changed successfully")
	}
	return nil
}

func (b *TempChannelBot) setCommandChannelHandler(context *CommandHandlerContext) error {
	if len(context.CommandArgs) > 1 {
		context.reply("Too many arguments, please check %vhelp to see how to use the command", context.ServerData.CommandPrefix())
		return nil
	}

	if len(context.CommandArgs) == 0 {
		if !context.ServerData.HasCommandChannelID() {
			context.reply("The custom command channel wasn't set yet, please check %vhelp to see how to use the command", context.ServerData.CommandPrefix())
			return nil
		}

		err := context.ServerData.ClearCommandChannelID()
		if err != nil {
			context.reply("An internal error has occurred")
			return fmt.Errorf("ClearCommandChannelID failed: %v", err)
		}

		context.reply("Removed specific command channel successfully")
		return nil
	} else if len(context.CommandArgs) == 1 {
		channelIDStr := context.CommandArgs[0]

		channelID, err := state.ParseDiscordID(channelIDStr)
		if err != nil {
			context.reply(`Invalid channel ID, please right click the channel and click "Copy ID"`)
			log.Printf("Invalid channel ID %q: %v", channelIDStr, err) // TODO: consider log level error
			return nil
		}

		if !context.channelExists(channelIDStr) {
			context.reply(`This channel doesn't exist, please right click the channel and click "Copy ID"`)
			return nil
		}

		if !context.textChannelExists(channelIDStr) {
			context.reply(`The requested channel isn't a text channel, please right click the channel and click "Copy ID"`)
			return nil
		}

		err = context.ServerData.SetCommandChannelID(channelID)
		if err != nil {
			context.reply("An internal error has occurred")
			return fmt.Errorf("SetCommandChannelID failed: %v", err)
		}

		context.reply("Specific command channel set successfully")
		return nil
	}

	return nil
}
