package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/faideww/chat-fishing/internal/bot"
	"github.com/faideww/chat-fishing/internal/fish"
	"github.com/faideww/chat-fishing/internal/ratelimit"
	"github.com/faideww/chat-fishing/internal/store"
)

func main() {
	config, err := LoadConfig()
	if err != nil {
		log.Fatal("failed to load config:", err)
	}

	reg, err := fish.LoadRegistryFromJSON(config.SpeciesJson)
	if err != nil {
		log.Fatal(err)
	}

	st, err := store.OpenSQLite(config.DBPath)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()

	session, err := discordgo.New("Bot " + config.DiscordToken)
	if err != nil {
		log.Fatal("failed to start session:", err)
	}

	session.ShardCount = config.ShardCount
	session.ShardID = config.ShardId

	if err := session.Open(); err != nil {
		log.Fatal("failed to open session connection:", err)
	}
	defer session.Close()

	appId := session.State.User.ID

	fishLim := ratelimit.NewLimiter(
		time.Duration(config.CooldownFishingMin)*time.Second,
		time.Duration(config.CooldownFishingMax)*time.Second,
		nil,
	)
	lbLim := ratelimit.NewLimiter(
		time.Duration(config.CooldownLeaderboardMin)*time.Second,
		time.Duration(config.CooldownLeaderboardMax)*time.Second,
		nil,
	)
	teardown, err := bot.Setup(session, appId, config.DevGuild, reg, st, fishLim, lbLim)
	if err != nil {
		log.Fatal("failed to setup bot:", err)
	}
	defer teardown()

	log.Println("Bot is running")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}
