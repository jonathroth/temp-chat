package integration_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/jonathroth/temp-chat/bot"
	"github.com/jonathroth/temp-chat/consts"
	"github.com/jonathroth/temp-chat/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

var IDRegex = regexp.MustCompile(`<#(\d+)>`)

/*
IntegrationTestSuite runs all bot commands using two bot clients.
To run the integ tests, you must first define the $INTEG_TESTS env variable.

It requires a total of 4 accounts with 4 access tokens, accessed through env variables:

1. The bot itself - $DISCORD_TOKEN
2. Admin accounts - $ADMIN_TOKEN - Used to create the different channels/categories.
3. Client #1 - CLIENT1_TOKEN
4. Client #2 - CLIENT2_TOKEN

The suite assumes all bots are in a single server, and that the admin bot has the Administrator privilege.
*/
type IntegrationTestSuite struct {
	suite.Suite

	bot            *TestSession
	provider       *MemoryDataProvider
	store          state.ServerStore
	tempChannelBot *bot.TempChannelBot

	admin   *TestSession
	client1 *TestSession
	client2 *TestSession

	server      *discordgo.Guild
	textChannel *discordgo.Channel

	cleanups []func()
}

func TestIntegrationTestSuite(t *testing.T) {
	_, exists := os.LookupEnv("INTEG_TESTS")
	if !exists {
		t.Skip("INTEG_TESTS is not defined, skipping")
	}
	suite.Run(t, &IntegrationTestSuite{})
}

func (s *IntegrationTestSuite) SetupSuite() {
	var err error
	s.admin = NewTestClientSession(s.T(), os.Getenv("INTEG_TEST_ADMIN_TOKEN"))
	s.client1 = NewTestClientSession(s.T(), os.Getenv("INTEG_TEST_CLIENT1_TOKEN"))
	s.client2 = NewTestClientSession(s.T(), os.Getenv("INTEG_TEST_CLIENT2_TOKEN"))

	servers, err := s.admin.UserGuilds(1, "", "")
	failOnErr(s.T(), err, "Failed getting servers")

	if !assert.Len(s.T(), servers, 1, "More than 1 guild found") {
		s.T().FailNow()
	}

	server := servers[0]
	s.server, err = s.admin.Guild(server.ID)
	failOnErr(s.T(), err, "Failed getting server")
}

func (s *IntegrationTestSuite) TearDownSuite() {
	assert.NoError(s.T(), s.admin.Close(), "Admin session Close() returned error")
	assert.NoError(s.T(), s.client1.Close(), "Client1 session Close() returned error")
	assert.NoError(s.T(), s.client2.Close(), "Client2 session Close() returned error")
}

func (s *IntegrationTestSuite) SetupTest() {
	s.bot = NewTestBotSession(s.T(), os.Getenv("INTEG_TEST_BOT_TOKEN"))

	var err error
	s.provider = NewMemoryDataProvider()
	s.store, err = state.NewSyncServerStore(s.provider)
	failOnErr(s.T(), err, "Failed initializing server store")
	s.tempChannelBot, err = bot.NewTempChannelBot(s.bot.Session, s.store)
	failOnErr(s.T(), err, "Failed initializing bot")

	s.tempChannelBot.AllowBots = true

	s.cleanups = []func(){}
	s.cleanups = append(s.cleanups, s.bot.AddHandler(s.tempChannelBot.MessageCreate))
	s.cleanups = append(s.cleanups, s.bot.AddHandler(s.tempChannelBot.ChannelDelete))
	s.cleanups = append(s.cleanups, s.bot.AddHandler(s.tempChannelBot.VoiceStatusUpdate))

	err = s.bot.Open()
	if !assert.NoError(s.T(), err, "Failed creating discord session") {
		s.T().FailNow()
	}

	s.textChannel = s.createChannel("command-channel", discordgo.ChannelTypeGuildText)
}

func (s *IntegrationTestSuite) TearDownTest() {
	for _, cleanupFunc := range s.cleanups {
		cleanupFunc()
	}

	s.deleteChannel(s.textChannel)
	s.tempChannelBot.CleanChannels()
	assert.NoError(s.T(), s.bot.Close(), "Bot session Close() returned error")
}

func (s *IntegrationTestSuite) TestMkch() {
	category := s.createChannel("category", discordgo.ChannelTypeGuildCategory)
	defer s.deleteChannel(category)

	err := s.admin.ChannelPermissionSet(category.ID, s.bot.Me.ID, consts.PermissionTypeMember, discordgo.PermissionManageChannels, 0)
	failOnErr(s.T(), err, "Failed giving temp-bot permissions")

	setupCommand := fmt.Sprintf("!setup %v", category.ID)
	s.admin.Command(s.textChannel.ID, setupCommand, s.bot.Me, "Server was setup successfully")

	voiceChannel1 := s.createChannel("voice1", discordgo.ChannelTypeGuildVoice)
	defer s.deleteChannel(voiceChannel1)
	voiceChannel2 := s.createChannel("voice2", discordgo.ChannelTypeGuildVoice)
	defer s.deleteChannel(voiceChannel2)

	voiceConn1, err := s.client1.ChannelVoiceJoin(s.server.ID, voiceChannel1.ID, true, true)
	failOnErr(s.T(), err, "Failed joining voice chat")

	response := s.client1.Command(s.textChannel.ID, "!mkch", s.bot.Me, "temporary channel was created")
	s.T().Logf("response: %q", response.Content)
	submatches := IDRegex.FindStringSubmatch(response.Content)
	if !s.Len(submatches, 2, "Expected 1 submatch") {
		return
	}

	tempChatID := submatches[1]
	_, err = s.admin.State.Channel(tempChatID)
	failOnErr(s.T(), err, "Created temp chat not found")

	if !s.True(s.client1.HasPermissions(tempChatID, discordgo.PermissionViewChannel), "No read permissions for tempchat creator") {
		return
	}
	if !s.False(s.client2.HasPermissions(tempChatID, discordgo.PermissionViewChannel), "User outside vc has permissions for tempchat") {
		return
	}

	content := "hi"
	s.client1.SendMessage(tempChatID, content)

	voiceConn2, err := s.client2.ChannelVoiceJoin(s.server.ID, voiceChannel1.ID, true, true)
	failOnErr(s.T(), err, "Failed joining voice chat")

	messages, err := s.client2.ChannelMessages(tempChatID, 1, "", "", "")
	failOnErr(s.T(), err, "Failed getting messages from text chat the bot is in")
	for _, message := range messages {
		assert.NotEqual(s.T(), content, message.Content, "Didn't expect seeing message sent before joining")
	}

	if !s.True(s.client1.HasPermissions(tempChatID, discordgo.PermissionViewChannel), "No read permissions for tempchat creator") {
		return
	}
	if !s.True(s.client2.HasPermissions(tempChatID, discordgo.PermissionViewChannel), "User didn't get read permissions") {
		return
	}

	err = voiceConn1.Disconnect()
	assert.NoError(s.T(), err, "Disconnecting from VC failed")
	err = voiceConn2.Disconnect()
	assert.NoError(s.T(), err, "Disconnecting from VC failed")

	_, err = s.client1.Channel(tempChatID)
	assert.Error(s.T(), err, "Channel wasn't removed after both users left")
	_, err = s.admin.Channel(tempChatID)
	assert.Error(s.T(), err, "Channel wasn't removed after both users left")

	voiceConn1, err = s.client1.ChannelVoiceJoin(s.server.ID, voiceChannel1.ID, true, true)
	failOnErr(s.T(), err, "Failed joining voice chat")
	defer func() {
		err = voiceConn1.Disconnect()
		assert.NoError(s.T(), err, "Disconnecting from VC failed")
	}()

	_, err = s.client1.Channel(tempChatID)
	assert.Error(s.T(), err, "Channel wasn't removed after both users left")
}

func (s *IntegrationTestSuite) TestHelp() {
	s.client1.Command(s.textChannel.ID, "!help", s.bot.Me, "TempChat is a bot that creates temporary text channels for Discord voice chats")
}

func (s *IntegrationTestSuite) TestSetupRequired() {
	mkchResponse := s.client1.Command(s.textChannel.ID, "!mkch", s.bot.Me, "bot hasn't been set up yet")

	serverID, err := state.ParseDiscordID(mkchResponse.GuildID)
	failOnErr(s.T(), err, "Failed to parse server ID for response")

	_, inDatabase := s.provider.database[serverID]
	assert.False(s.T(), inDatabase, "Message in database before setup")

	category := s.createChannel("temp", discordgo.ChannelTypeGuildCategory)
	defer s.deleteChannel(category)

	err = s.admin.ChannelPermissionSet(category.ID, s.bot.Me.ID, consts.PermissionTypeMember, discordgo.PermissionManageChannels, 0)
	failOnErr(s.T(), err, "Failed giving temp-bot permissions")

	s.admin.Command(s.textChannel.ID, fmt.Sprintf("!setup %v", category.ID), s.bot.Me, "Server was setup successfully")

	data, inDatabase := s.provider.database[serverID]
	assert.True(s.T(), inDatabase, "Message not in database after setup")
	assert.Equal(s.T(), data.TempChannelCategoryID().RESTAPIFormat(), category.ID, "Category not equal to setup category")

	s.client1.Command(s.textChannel.ID, "!mkch", s.bot.Me, "You must be in a voice chat to use this command")

	category2 := s.createChannel("temp2", discordgo.ChannelTypeGuildCategory)
	defer s.deleteChannel(category2)

	err = s.admin.ChannelPermissionSet(category2.ID, s.bot.Me.ID, consts.PermissionTypeMember, discordgo.PermissionManageChannels, 0)
	failOnErr(s.T(), err, "Failed giving temp-bot permissions")

	s.admin.Command(s.textChannel.ID, fmt.Sprintf("!setup %v", category2.ID), s.bot.Me, "Server was setup successfully")

	data, inDatabase = s.provider.database[serverID]
	assert.True(s.T(), inDatabase, "Message not in database after setup")
	assert.Equal(s.T(), data.TempChannelCategoryID().RESTAPIFormat(), category2.ID, "Category wasn't updated")
}

func (s *IntegrationTestSuite) TestAdminOnly() {
	s.client1.Command(s.textChannel.ID, "!setup", s.bot.Me, `must have "Administrator" permissions`)
	s.admin.Command(s.textChannel.ID, "!setup", s.bot.Me, "Missing category ID, please check !help to see how to use the command")
}

func (s *IntegrationTestSuite) TestDM() {
	s.T().Skip(`Bots cannot DM other bots, test results in HTTP 403 Forbidden, {"message": "Cannot send messages to this user", "code": 50007}`)
	botDMChannel, err := s.client1.UserChannelCreate(s.bot.Me.ID)
	failOnErr(s.T(), err, "Failed to create DM channel with bot")
	s.client1.Command(botDMChannel.ID, "!help", s.bot.Me, "TempChat is a bot that creates temporary text channels for Discord voice chats")
	s.client1.Command(botDMChannel.ID, "help", s.bot.Me, "bot doesn't accept any command besides !help in private messages")
	s.client1.Command(botDMChannel.ID, "....", s.bot.Me, "bot doesn't accept any command besides !help in private messages")
}

func (s *IntegrationTestSuite) createChannel(name string, channelType discordgo.ChannelType) *discordgo.Channel {
	channel, err := s.admin.GuildChannelCreate(s.server.ID, name, channelType)
	failOnErr(s.T(), err, "Failed creating command channel")
	return channel
}

func (s *IntegrationTestSuite) deleteChannel(channel *discordgo.Channel) {
	_, err := s.admin.ChannelDelete(channel.ID)
	assert.NoError(s.T(), err, "Couldn't remove category")
}
