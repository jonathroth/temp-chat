package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/jonathroth/temp-chat/bot"
	"github.com/jonathroth/temp-chat/state"
)

var (
	discordToken = os.Getenv("DISCORD_TOKEN")
	postgresAddr = os.Getenv("DATABASE_URL")
)

func main() {
	if discordToken == "" {
		log.Fatalf("A discord token is required to run")
	}

	if postgresAddr == "" {
		log.Fatalf("A PostgreSQL database URL is required to run")
	}

	session, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		log.Fatalf("Failed initializing discord connection: %v", err)
	}

	serversProvider, err := state.NewPostgresServersProvider(postgresAddr)
	if err != nil {
		log.Fatalf("Failed connecting to the database: %v", err)
	}

	store, err := state.NewSyncServerStore(serversProvider)
	if err != nil {
		log.Fatalf("Failed initializing server store: %v", err)
	}

	tempChannelBot, err := bot.NewTempChannelBot(session, store)
	if err != nil {
		log.Fatalf("Failed initializing bot: %v", err)
	}

	waitForBot(session, tempChannelBot)

	err = session.Close()
	if err != nil {
		log.Fatalf("Bot ClosE() failed: %v", err)
	}
}

func waitForBot(session *discordgo.Session, tempChannelBot *bot.TempChannelBot) {
	defer tempChannelBot.CleanChannels()

	session.AddHandler(tempChannelBot.MessageCreate)
	session.AddHandler(tempChannelBot.ChannelDelete)
	session.AddHandler(tempChannelBot.VoiceStatusUpdate)

	err := session.Open()
	if err != nil {
		log.Fatalf("Failed connecting to discord: %v", err)
	}

	log.Println("Bot is now running...")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	exitSignal := <-sc
	log.Printf("Got signal %v, exiting", exitSignal.String())
}
