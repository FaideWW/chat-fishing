package bot

import "github.com/bwmarrin/discordgo"

func commandDefs() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{Name: "fish", Description: "Cast a line"},
		{
			Name:        "leaderboard",
			Description: "Show the biggest catches",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "species",
					Description: "Filter by species key",
					Required:    false,
				},
			},
		},
	}
}
