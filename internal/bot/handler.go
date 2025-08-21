package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/faideww/chat-fishing/internal/fish"
	"github.com/faideww/chat-fishing/internal/ratelimit"
	"github.com/faideww/chat-fishing/internal/store"
)

type module struct {
	s          *discordgo.Session
	appId      string
	scopeGuild string
	picker     *fish.Picker
	reg        *fish.Registry
	fishLim    *ratelimit.Limiter
	lbLim      *ratelimit.Limiter
	store      *store.SQLiteStore
}

func Setup(
	session *discordgo.Session,
	appId, scopeGuild string,
	reg *fish.Registry,
	store *store.SQLiteStore,
	fishLim *ratelimit.Limiter,
	lbLim *ratelimit.Limiter,
) (func(), error) {

	picker := fish.NewPicker(reg, nil)
	m := &module{
		s:          session,
		appId:      appId,
		scopeGuild: scopeGuild,
		picker:     picker,
		reg:        reg,
		store:      store,
		fishLim:    fishLim,
		lbLim:      lbLim,
	}

	cmds := commandDefs()

	created, err := session.ApplicationCommandBulkOverwrite(appId, scopeGuild, cmds)
	if err != nil {
		return nil, fmt.Errorf("failed to register commands: %w", err)
	}

	for _, c := range created {
		fmt.Printf("command active: %s (%s)\n", c.Name, c.Description)
	}

	session.AddHandler(m.onInteraction)

	return func() {}, nil
}

func (m *module) onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	switch i.ApplicationCommandData().Name {
	case "fish":
		m.handleFish(s, i)
	case "leaderboard":
		m.handleLeaderboard(s, i)
	}
}

func (m *module) handleFish(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Validate execution context (/fish must be run in a server)
	if i.GuildID == "" {
		respondEphemeral(s, i, "Use this command in a server!")
		return
	}

	userIdStr := ""
	if i.Member != nil && i.Member.User != nil {
		userIdStr = i.Member.User.ID
	} else if i.User != nil {
		userIdStr = i.User.ID
	}

	// Rate limiting
	if ok, rem := m.fishLim.Try(i.GuildID, userIdStr); !ok {
		respondEphemeral(s, i, fmt.Sprintf("⏳ You’re reeling in… try again in %s.", pretty(rem)))
		return
	}

	// Send a deferred ack so we don't hit a timeout while processing
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		logREST("defer response failed", err)
		return
	}

	// Roll fish
	catchId := m.picker.PickId()
	sz := m.picker.RollSize(catchId)

	err := m.store.Add(context.TODO(), fish.Catch{
		GuildId:   toInt64(i.GuildID),
		UserId:    toInt64(userIdStr),
		SpeciesId: catchId,
		Size:      sz,
		CaughtAt:  time.Now(),
	})

	if err != nil {
		logREST("failed to insert", err)
	}

	tier := m.picker.SpeciesTier(catchId)
	sp, _ := m.reg.GetById(catchId)
	szClass := fish.SizeClassFor(sp, sz)

	username := i.Member.Nick
	if username == "" {
		username = i.Member.User.Username
	}

	// TODO: some words beginning with consonants use 'an' (hour, heir, honest).
	indefArticle := "a"
	if sp.Name[0] == 'a' || sp.Name[0] == 'e' || sp.Name[0] == 'i' || sp.Name[0] == 'o' || sp.Name[0] == 'u' {
		indefArticle = "an"
	}

	thumb := m.reg.EmbedThumb(fish.SpeciesId(sp.Id))
	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s caught %s %s!", username, indefArticle, sp.Name),
		Description: fmt.Sprintf("Size: **%.1f cm**  ·  **%s**\nRarity: **%s**", sz, szClass.String(), tier.String()),
		Color:       fish.ColorForTier(tier),
		Thumbnail:   thumb,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Tip: Bigger fish are rarer!",
		},
	}

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	}); err != nil {
		logREST("edit failed", err)
	}
}

func (m *module) handleLeaderboard(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Validate execution context (/leaderboard must be run in a server)
	if i.GuildID == "" {
		respondEphemeral(s, i, "Use this command in a server!")
		return
	}

	// Rate limiting
	if ok, rem := m.lbLim.Try(i.GuildID, "leaderboard"); !ok {
		respondEphemeral(s, i, fmt.Sprintf("⏳ Leaderboard refreshing... try again in %s.", pretty(rem)))
		return
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		logREST("defer response failed", err)
		return
	}

	data := i.ApplicationCommandData()
	speciesId := fish.SpeciesId(-1)
	limit := 10
	for _, opt := range data.Options {
		switch opt.Name {
		case "species":
			fishKey := opt.StringValue()
			var ok bool
			speciesId, ok = m.reg.IdByKey(fishKey)
			if !ok {
				respondEphemeral(s, i, fmt.Sprintf("Unknown fish '%s'", fishKey))
				return
			}
		}
	}

	var rows []fish.Catch
	if speciesId >= 0 {
		rs, err := m.store.TopBySizeGuildSpecies(context.TODO(), toInt64(i.GuildID), speciesId, limit)
		if err != nil {
			editResponseText(s, i, "Error loading leaderboard.")
			fmt.Printf("error: %v", err)
			return
		}
		rows = rs
	} else {
		rs, err := m.store.TopBySize(context.TODO(), toInt64(i.GuildID), limit)
		if err != nil {
			editResponseText(s, i, "Error loading leaderboard.")
			fmt.Printf("error: %v", err)
			return
		}
		rows = rs
	}

	if len(rows) == 0 {
		editResponseText(s, i, "No catches yet - type `/fish` to make the first!")
		return
	}

	desc := strings.Builder{}

	for idx, c := range rows {
		pos := idx + 1
		// mention format: <@USERID>
		uid := fmt.Sprintf("%d", c.UserId)
		sp, _ := m.reg.GetById(fish.SpeciesId(c.SpeciesId))
		szClass := fish.SizeClassFor(sp, c.Size)
		line := fmt.Sprintf("**#%d** **%.1f cm (%s)** — <@%s> — %s\n",
			pos, c.Size, szClass.String(), uid, sp.Name)
		desc.WriteString(line)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🏆 Leaderboard - Biggest Catches",
		Description: desc.String(),
		Color:       0xf1c40f,
	}

	if speciesId >= 0 {
		if sp, ok := m.reg.GetById(fish.SpeciesId(speciesId)); ok {
			embed.Title = fmt.Sprintf("🏆 Leaderboard — %s", sp.Name)
		}
	}

	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}

func respondEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func editResponseText(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
}

func pretty(d time.Duration) string {
	// mm:ss
	if d < 0 {
		d = 0
	}
	m := int(d / time.Minute)
	s := int((d % time.Minute) / time.Second)
	return fmt.Sprintf("%d:%02d", m, s)
}

func logREST(msg string, err error) {
	if rerr, ok := err.(*discordgo.RESTError); ok && rerr.Message != nil {
		log.Printf("%s: code=%d msg=%s", msg, rerr.Message.Code, rerr.Message.Message)
	} else {
		log.Printf("%s: %v", msg, err)
	}
}

// Convert snowflake string to int64 (you said you switched DB to int64s)
func toInt64(snowflake string) int64 {
	n, _ := strconv.ParseInt(snowflake, 10, 64)
	return n
}
