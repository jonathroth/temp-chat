package integration_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/jonathroth/temp-chat/bot"
	"github.com/jonathroth/temp-chat/consts"
	"github.com/jonathroth/temp-chat/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

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

	store, err := state.NewSyncServerStore(NewMemoryDataProvider())
	failOnErr(s.T(), err, "Failed initializing server store")
	s.tempChannelBot, err = bot.NewTempChannelBot(s.bot.Session, store)
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

	s.textChannel, err = s.admin.GuildChannelCreate(s.server.ID, "command-channel", discordgo.ChannelTypeGuildText)
	failOnErr(s.T(), err, "Failed creating command channel")
}

func (s *IntegrationTestSuite) TearDownTest() {
	for _, cleanupFunc := range s.cleanups {
		cleanupFunc()
	}

	_, err := s.admin.ChannelDelete(s.textChannel.ID)
	assert.NoError(s.T(), err, "Failed to delete command channel")

	assert.NoError(s.T(), s.bot.Close(), "Bot session Close() returned error")
}

func (s *IntegrationTestSuite) TestSetupRequired() {
	s.client1.Command(s.textChannel.ID, "!mkch", s.bot.Me, "bot hasn't been set up yet")

	category, err := s.admin.GuildChannelCreate(s.server.ID, "temp", discordgo.ChannelTypeGuildCategory)
	failOnErr(s.T(), err, "Failed creating category")

	defer func() {
		_, err := s.admin.ChannelDelete(category.ID)
		assert.NoError(s.T(), err, "Couldn't remove category")
	}()

	err = s.admin.ChannelPermissionSet(category.ID, s.bot.Me.ID, consts.PermissionTypeMember, discordgo.PermissionManageChannels, 0)
	failOnErr(s.T(), err, "Failed giving temp-bot permissions")

	setupCommand := fmt.Sprintf("!setup %v", category.ID)
	s.admin.Command(s.textChannel.ID, setupCommand, s.bot.Me, "Server was setup successfully")

	s.client1.Command(s.textChannel.ID, "!mkch", s.bot.Me, "You must be in a voice chat to use this command")
}

func (s *IntegrationTestSuite) TestAdminOnly() {
	s.client1.Command(s.textChannel.ID, "!setup", s.bot.Me, `must have "Administrator" permissions`)
	s.admin.Command(s.textChannel.ID, "!setup", s.bot.Me, "Missing category ID, please check !help to see how to use the command")
}
