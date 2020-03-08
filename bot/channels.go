package bot

import (
	"errors"
	"log"

	"github.com/Pallinder/go-randomdata"
	"github.com/bwmarrin/discordgo"
	"github.com/jonathroth/temp-chat/consts"
	"github.com/jonathroth/temp-chat/state"
)

// ChannelDelete is called whenever a channel is deleted in a server the bot is in.
func (b *TempChannelBot) ChannelDelete(s *discordgo.Session, m *discordgo.ChannelDelete) {
	channelID, err := state.ParseDiscordID(m.ID)
	if err != nil {
		log.Fatalf("Failed to parse channel ID of a channel that was just deleted")
	}

	if m.Type == discordgo.ChannelTypeGuildVoice {
		tempChannel, exists := b.tempChannels.GetTempChannelForVoiceChat(channelID)
		if exists {
			log.Printf("An administrator deleted the voice channel for temp chat %v", tempChannel.channelID)
			b.tempChannels.DeleteTempChannel(tempChannel)
		}
	} else if m.Type == discordgo.ChannelTypeGuildText {
		tempChannel, exists := b.tempChannels.GetTempChannelByID(channelID)
		if exists {
			log.Printf("An administrator deleted the temp chat %v", tempChannel.channelID)
			b.tempChannels.DeleteTempChannel(tempChannel)
		}
	}
}

// VoiceStatusUpdate is called whenever a user joins/leaves/moves a voice channel.
func (b *TempChannelBot) VoiceStatusUpdate(s *discordgo.Session, vsu *discordgo.VoiceStateUpdate) {
	userID, err := state.ParseDiscordID(vsu.UserID)
	if err != nil {
		log.Fatalf("Failed to parse user ID from user voice status update: %v", err)
	}

	if vsu.ChannelID == "" {
		// User has left voice chat
		err = b.tempChannels.RemoveUserFromChannel(userID)
		if err != nil {
			log.Printf("Failed to assign user to new temp channel: %v", err) // TODO: notify user somehow?
		}
		return
	}

	// User joined voice chat/switch to another chat
	voiceChannelID, err := state.ParseDiscordID(vsu.ChannelID)
	if err != nil {
		log.Fatalf("Failed to parse channel ID from user voice status update: %v", err)
	}

	err = b.tempChannels.AssignUserToTempChannel(userID, voiceChannelID)
	if err != nil {
		log.Printf("Failed to assign user to new temp channel: %v", err) // TODO: notify user somehow?
	}
}

type channelMap map[state.DiscordID]*TempChannel

// TempChannelList manages the list of temporary channels created by the bot.
type TempChannelList struct {
	tempChannelIDToTempChannel  channelMap
	voiceChannelIDToTempChannel channelMap
	userIDToTempChannel         channelMap

	session *discordgo.Session
}

// NewTempChannelList initializes a new instance of TempChannelList
func NewTempChannelList(session *discordgo.Session) *TempChannelList {
	return &TempChannelList{
		tempChannelIDToTempChannel:  channelMap{},
		voiceChannelIDToTempChannel: channelMap{},
		userIDToTempChannel:         channelMap{},
		session:                     session,
	}
}

// GetTempChannelForVoiceChat returns the temporary text channel that is assigned to the given voice channel ID.
func (l *TempChannelList) GetTempChannelForVoiceChat(voiceChannelID state.DiscordID) (*TempChannel, bool) {
	tempChannel, found := l.voiceChannelIDToTempChannel[voiceChannelID]
	return tempChannel, found
}

// GetTempChannelByID returns the temporary text channel that has the given ID.
func (l *TempChannelList) GetTempChannelByID(tempChannelID state.DiscordID) (*TempChannel, bool) {
	tempChannel, found := l.tempChannelIDToTempChannel[tempChannelID]
	return tempChannel, found
}

// AddTempChannel adds a new temp channel to the list.
func (l *TempChannelList) AddTempChannel(tempChannel *TempChannel) {
	l.tempChannelIDToTempChannel[tempChannel.channelID] = tempChannel
	l.voiceChannelIDToTempChannel[tempChannel.voiceChannelID] = tempChannel
	for userID := range tempChannel.members {
		l.userIDToTempChannel[userID] = tempChannel
	}
}

// DeleteTempChannel deletes a temp channel from the list.
func (l *TempChannelList) DeleteTempChannel(tempChannel *TempChannel) {
	for userID := range tempChannel.members {
		delete(l.userIDToTempChannel, userID)
	}

	delete(l.tempChannelIDToTempChannel, tempChannel.channelID)
	delete(l.voiceChannelIDToTempChannel, tempChannel.voiceChannelID)

	_, err := l.session.State.Channel(tempChannel.channelID.RESTAPIFormat())
	if existsInState(err) {
		err := tempChannel.Delete()
		if err != nil {
			// TODO: check for permission error, notify the server
			log.Fatalf("Failed to delete temp channel: %v", err)
		}
	}
}

// AssignUserToTempChannel gives a user access to a temp voice channel.
// It will remove access from a previous chat, if the user was in one.
func (l *TempChannelList) AssignUserToTempChannel(userID state.DiscordID, voiceChannelID state.DiscordID) error {
	err := l.RemoveUserFromChannel(userID)
	if err != nil {
		return err
	}

	tempChannel, found := l.voiceChannelIDToTempChannel[voiceChannelID]
	if !found {
		// Channel doesn't have a temp chat, do nothing
		return nil
	}

	err = tempChannel.AllowUserAccess(userID)
	if err != nil {
		return err
	}

	l.userIDToTempChannel[userID] = tempChannel
	return nil
}

// RemoveUserFromChannel removes a user when from a voice chat when the user.
func (l *TempChannelList) RemoveUserFromChannel(userID state.DiscordID) error {
	oldChannel, found := l.userIDToTempChannel[userID]
	if found {
		channelEmpty, err := oldChannel.DenyUserAccess(userID)
		if err != nil {
			return err
		}

		if channelEmpty {
			l.DeleteTempChannel(oldChannel)
		}
	}

	delete(l.userIDToTempChannel, userID)
	return nil
}

// TempChannel is a temporary text channel created by the bot.
type TempChannel struct {
	channelID      state.DiscordID
	voiceChannelID state.DiscordID

	channel *discordgo.Channel

	// Value isn't used, map is used for faster checks
	members map[state.DiscordID]bool

	session *discordgo.Session
}

// NewTempChannel creates a temporary channel for the given users.
func NewTempChannel(context *CommandHandlerContext, voiceChannelID state.DiscordID, userIDs []state.DiscordID) (*TempChannel, error) {
	channel, err := createTempChannel(context, userIDs)
	if err != nil {
		return nil, err
	}

	channelID, err := state.ParseDiscordID(channel.ID)
	if err != nil {
		return nil, err
	}

	userIDsMap := map[state.DiscordID]bool{}
	for _, userID := range userIDs {
		userIDsMap[userID] = true
	}

	return &TempChannel{
		channelID:      channelID,
		voiceChannelID: voiceChannelID,
		channel:        channel,
		members:        userIDsMap,
		session:        context.Session,
	}, nil
}

func createTempChannel(context *CommandHandlerContext, userIDs []state.DiscordID) (*discordgo.Channel, error) {
	everyoneRoleID, err := getEveryoneRoleID(context)
	if err != nil {
		return nil, err
	}

	overwrites := []*discordgo.PermissionOverwrite{
		&discordgo.PermissionOverwrite{
			ID:   everyoneRoleID,
			Type: consts.PermissionTypeRole,
			Deny: discordgo.PermissionReadMessages,
		},
	}

	for _, userID := range userIDs {
		perm := &discordgo.PermissionOverwrite{
			ID:    userID.RESTAPIFormat(),
			Type:  consts.PermissionTypeMember,
			Allow: discordgo.PermissionReadMessages,
		}
		overwrites = append(overwrites, perm)
	}

	creationData := discordgo.GuildChannelCreateData{
		Name:                 randomdata.SillyName(),
		Type:                 discordgo.ChannelTypeGuildText,
		PermissionOverwrites: overwrites,
		ParentID:             context.ServerData.TempChannelCategoryID().RESTAPIFormat(),
	}

	return context.Session.GuildChannelCreateComplex(context.Event.GuildID, creationData)
}

func getEveryoneRoleID(context *CommandHandlerContext) (string, error) {
	guild, err := context.Session.State.Guild(context.Event.GuildID)
	if err != nil {
		return "", nil
	}

	for _, role := range guild.Roles {
		if role.Name == consts.EveryoneRoleName {
			return role.ID, nil
		}
	}

	return "", errors.New("@everyone not found")
}

// AllowUserAccess gives a user access to the temporary channel.
func (c *TempChannel) AllowUserAccess(userID state.DiscordID) error {
	_, userIsChannelMember := c.members[userID]
	if userIsChannelMember {
		log.Printf("User %v is already in the channel %v", userID, c.channel.Name)
	}

	err := c.session.ChannelPermissionSet(c.channel.ID, userID.RESTAPIFormat(), consts.PermissionTypeMember, discordgo.PermissionReadMessages, 0)
	if err != nil {
		return err
	}

	c.members[userID] = true
	return nil
}

// DenyUserAccess removes the user's permission to read the channel.
//
// Returns whether the channel is empty.
func (c *TempChannel) DenyUserAccess(userID state.DiscordID) (bool, error) {
	err := c.session.ChannelPermissionDelete(c.channel.ID, userID.RESTAPIFormat())
	if err != nil {
		return false, err
	}

	delete(c.members, userID)
	if len(c.members) > 0 {
		return false, nil
	}

	return true, nil
}

// Delete deletes the temporary channel.
func (c *TempChannel) Delete() error {
	_, err := c.session.ChannelDelete(c.channel.ID)
	if err != nil {
		return err
	}

	return nil
}
