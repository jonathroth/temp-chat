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

	store := state.NewPostgresServerStore(postgresAddr)
	err = store.Connect()
	if err != nil {
		log.Fatalf("Failed connecting to the database: %v", err)
	}

	tempChannelBot, err := bot.NewTempChannelBot(session, store)
	if err != nil {
		log.Fatalf("Failed initializing bot: %v", err)
	}
	session.AddHandler(tempChannelBot.MessageCreate)

	err = session.Open()
	if err != nil {
		log.Fatalf("Failed connecting to discord: %v", err)
	}

	log.Println("Bot is now running...")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	exitSignal := <-sc
	log.Printf("Got signal %v, exiting", exitSignal.String())

	session.Close()
}
