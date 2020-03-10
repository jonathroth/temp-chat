package integration_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jonathroth/temp-chat/consts"
	"github.com/stretchr/testify/assert"
)

func failOnErr(t *testing.T, err error, message string) {
	if !assert.NoError(t, err, message) {
		t.FailNow()
	}
}

type TestSession struct {
	*discordgo.Session
	Me *discordgo.User
	t  *testing.T
}

func NewTestClientSession(t *testing.T, token string) *TestSession {
	session := NewTestBotSession(t, token)
	session.State.MaxMessageCount = 10

	err := session.Open()
	failOnErr(t, err, "Failed to open discord session")

	return session
}

// NewTestBotSession creates a discord session without opening it, so we can config the handlers first
func NewTestBotSession(t *testing.T, token string) *TestSession {
	discordSession, err := discordgo.New("Bot " + token)
	failOnErr(t, err, "Failed to create discord session")

	me, err := discordSession.User("@me")
	failOnErr(t, err, "Failed to get @me")

	session := &TestSession{
		Session: discordSession,
		Me:      me,
		t:       t,
	}
	return session
}

func (s *TestSession) Command(channelID string, command string, responder *discordgo.User, responseContains string) {
	m := s.SendMessage(channelID, command)
	_ = s.ExpectResponse(m, responder, responseContains, 5*time.Second)
}

func (s *TestSession) CommandTimeout(channelID string, command string, responder *discordgo.User, responseContains string, within time.Duration) {
	m := s.SendMessage(channelID, command)
	_ = s.ExpectResponse(m, responder, responseContains, within)
}

func (s *TestSession) SendMessage(channelID string, message string, args ...interface{}) *discordgo.Message {
	sentMessage, err := s.ChannelMessageSend(channelID, fmt.Sprintf(message, args...))
	failOnErr(s.t, err, "Failed to send message")
	return sentMessage
}

func (s *TestSession) ExpectResponse(to *discordgo.Message, from *discordgo.User, textContains string, within time.Duration) *discordgo.Message {
	pollInterval := 100 * time.Millisecond

	if within < pollInterval {
		within = 1 * time.Second
	}

	interval := time.Tick(pollInterval)
	timeout := time.After(within)

	s.t.Logf("Waiting for response %q for %v", textContains, within)
	start := time.Now()

	for {
		select {
		case <-timeout:
			failOnErr(s.t, errors.New("response timed out"), "Failed getting response within timeout")
			return nil
		case <-interval:
			channel, err := s.State.Channel(to.ChannelID)
			failOnErr(s.t, err, "Failed to get channel")
			for _, message := range channel.Messages {
				if message.Author.ID == from.ID && strings.Contains(message.Content, textContains) {
					s.t.Logf("Got response after %v", time.Since(start))
					return message
				}
			}
		}
	}
}

func (s *TestSession) HasPermissions(server *discordgo.Guild, channel *discordgo.Channel, permission int) bool {
	user, err := s.Session.GuildMember(server.ID, s.Me.ID)
	failOnErr(s.t, err, "Couldn't get user")

	roles := []*discordgo.Role{}
	for _, role := range server.Roles {
		if role.Name == consts.EveryoneRoleName {
			roles = append(roles, role)
		}

		for _, userRoleID := range user.Roles {
			if role.ID == userRoleID {
				roles = append(roles, role)
			}
		}
	}

	// Check permission overrides denying access to specific channel
	for _, overwrite := range channel.PermissionOverwrites {
		if overwrite.Type == consts.PermissionTypeMember && overwrite.ID == s.Me.ID && overwrite.Deny&permission != 0 {
			return false
		}
		for _, role := range roles {
			if overwrite.Type == consts.PermissionTypeRole && overwrite.ID == role.ID && overwrite.Deny&permission != 0 {
				return false
			}
		}
	}

	// Check global role
	for _, role := range roles {
		if role.Permissions&permission == 0 {
			return false
		}
	}

	return true
}
