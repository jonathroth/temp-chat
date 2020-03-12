package integration_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
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

func (s *TestSession) Command(channelID string, command string, responder *discordgo.User, responseContains string) *discordgo.Message {
	m := s.SendMessage(channelID, command)
	return s.ExpectResponse(m, responder, responseContains, 5*time.Second)
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

	channel, err := s.State.Channel(to.ChannelID)
	failOnErr(s.t, err, "Failed to get channel")

	s.t.Logf("Waiting for response %q for %v", textContains, within)
	start := time.Now()

	for {
		select {
		case <-timeout:
			failOnErr(s.t, errors.New("response timed out"), "Failed getting response within timeout")
			return nil
		case <-interval:
			message := s.findMessage(channel, from, textContains)
			if message != nil {
				s.t.Logf("Got response after %v", time.Since(start))
				return message
			}
		}
	}
}

func (s *TestSession) findMessage(channel *discordgo.Channel, from *discordgo.User, textContains string) *discordgo.Message {
	s.State.RLock()
	defer s.State.RUnlock()
	for _, message := range channel.Messages {
		if message.Author.ID == from.ID && strings.Contains(message.Content, textContains) {
			return message
		}
	}

	return nil
}

func (s *TestSession) HasPermissions(channel *discordgo.Channel, permission int) bool {
	s.RLock()
	defer s.RUnlock()
	permissions, err := s.Session.UserChannelPermissions(s.Me.ID, channel.ID)
	failOnErr(s.t, err, "Failed to get permissions")

	return permissions&permission != 0
}
