package bot

import (
	"fmt"
	"log"
	"strconv"

	"github.com/bwmarrin/discordgo"
)

func (b *TempChannelBot) replyToSenderAndLog(s *discordgo.Session, channelID string, message string, args ...interface{}) {
	log.Printf(message, args...)

	_, err := s.ChannelMessageSend(channelID, fmt.Sprintf(message, args...))
	if err != nil {
		log.Fatalf("Failed sending message response: %v", err)
	}
}

func parseID(id string) (uint64, error) {
	return strconv.ParseUint(id, 10, 64)
}

func formatID(id uint64) string {
	return strconv.FormatUint(id, 10)
}

func existsInState(err error) bool {
	return err != discordgo.ErrStateNotFound
}
