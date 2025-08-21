package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	SpeciesJson            string
	DiscordToken           string
	DevGuild               string
	DBPath                 string
	ShardCount             int
	ShardId                int
	CooldownFishingMin     int
	CooldownFishingMax     int
	CooldownLeaderboardMin int
	CooldownLeaderboardMax int
}

func LoadConfig() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load .env:", err)
	}
	speciesJson := os.Getenv("SPECIES_JSON")
	if speciesJson == "" {
		return nil, fmt.Errorf("No SPECIES_JSON in environment")
	}

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("No DISCORD_TOKEN in environment")
	}

	devGuild := os.Getenv("DEV_GUILD_ID")
	if devGuild == "" {
		return nil, fmt.Errorf("No DEV_GUILD_ID in environment")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		return nil, fmt.Errorf("No DB_PATH in environment")
	}

	shardCount, err := loadInt("SHARD_COUNT", 1)
	if err != nil {
		return nil, err
	}
	shardId, err := loadInt("SHARD_ID", 0)
	if err != nil {
		return nil, err
	}
	cooldownFishingMin, err := loadInt("COOLDOWN_FISHING_MIN", 240)
	if err != nil {
		return nil, err
	}
	cooldownFishingMax, err := loadInt("COOLDOWN_FISHING_MAX", 300)
	if err != nil {
		return nil, err
	}
	cooldownLeaderboardMin, err := loadInt("COOLDOWN_LEADERBOARD_MIN", 30)
	if err != nil {
		return nil, err
	}
	cooldownLeaderboardMax, err := loadInt("COOLDOWN_LEADERBOARD_MAX", 30)
	if err != nil {
		return nil, err
	}

	return &Config{
		SpeciesJson:            speciesJson,
		DiscordToken:           token,
		DevGuild:               devGuild,
		DBPath:                 dbPath,
		ShardCount:             shardCount,
		ShardId:                shardId,
		CooldownFishingMin:     cooldownFishingMin,
		CooldownFishingMax:     cooldownFishingMax,
		CooldownLeaderboardMin: cooldownLeaderboardMin,
		CooldownLeaderboardMax: cooldownLeaderboardMax,
	}, nil
}

func loadInt(key string, defValue int) (int, error) {
	value := os.Getenv(key)
	if value != "" {
		return strconv.Atoi(value)
	}

	return defValue, nil
}
