package fish

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
)

type SpeciesId int

type Species struct {
	Id       SpeciesId
	Key      string // stable string id if we decide to rename the fish for any reason
	Name     string
	Weight   int     // rarity weight (higher = more common)
	MinSize  float64 // in cm
	MaxSize  float64
	SizeBias float64 // 1.0 is uniform, >1 means larger is rarer
	Tags     []string
	Image    string
}

type SpeciesJSON struct {
	Id       int      `json:"id"`
	Key      string   `json:"key"`
	Name     string   `json:"name"`
	Weight   int      `json:"weight"`
	MinSize  float64  `json:"minSize"`
	MaxSize  float64  `json:"maxSize"`
	SizeBias float64  `json:"sizeBias"`
	Tags     []string `json:"tags"`
	Image    string   `json:"thumbnail"`
}

type Registry struct {
	byId  []Species
	byKey map[string]SpeciesId
}

func LoadRegistryFromJSON(path string) (*Registry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var arr []SpeciesJSON
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, err
	}
	if len(arr) == 0 {
		return nil, fmt.Errorf("species list is empty")
	}

	maxId := -1
	ids := make([]int, len(arr))
	seenKey := map[string]bool{}
	seenId := map[int]bool{}

	for i, sj := range arr {
		id := sj.Id
		if id < 0 {
			return nil, fmt.Errorf("negative id at index %d", i)
		}
		if seenId[id] {
			return nil, fmt.Errorf("duplicate id %d", id)
		}
		if sj.Key == "" {
			return nil, fmt.Errorf("missing key at id %d", id)
		}
		if seenKey[sj.Key] {
			return nil, fmt.Errorf("duplicate key %q", sj.Key)
		}

		seenId[id] = true
		seenKey[sj.Key] = true
		ids[i] = id
		if id > maxId {
			maxId = id
		}
	}

	byId := make([]Species, maxId+1)
	for i, sj := range arr {
		id := ids[i]
		if byId[id].Key != "" {
			return nil, fmt.Errorf("non-dense id assignment at %d", id)
		}

		if sj.Weight < 1 {
			sj.Weight = 1
		}
		byId[id] = Species{
			Id:       SpeciesId(id),
			Key:      sj.Key,
			Name:     sj.Name,
			Weight:   sj.Weight,
			MinSize:  sj.MinSize,
			MaxSize:  sj.MaxSize,
			SizeBias: sj.SizeBias,
			Tags:     sj.Tags,
		}
	}

	byKey := make(map[string]SpeciesId, len(arr))
	for id, sp := range byId {
		if sp.Key == "" {
			return nil, fmt.Errorf("gap at id %d", id)
		}
		byKey[sp.Key] = SpeciesId(id)
	}

	return &Registry{byId: byId, byKey: byKey}, nil
}

func (r *Registry) GetById(id SpeciesId) (Species, bool) {
	if int(id) < 0 || int(id) >= len(r.byId) {
		return Species{}, false
	}
	return r.byId[id], true
}

func (r *Registry) NameById(id SpeciesId) string {
	if sp, ok := r.GetById(id); ok {
		return sp.Name
	}
	return "Unknown"
}

func (r *Registry) IdByKey(key string) (SpeciesId, bool) {
	id, ok := r.byKey[key]
	return id, ok
}

func (r *Registry) EmbedThumb(id SpeciesId) *discordgo.MessageEmbedThumbnail {
	if sp, ok := r.GetById(id); ok && sp.Image != "" {
		return &discordgo.MessageEmbedThumbnail{URL: sp.Image}
	}
	return nil
}

func (r *Registry) All() []Species {
	out := make([]Species, len(r.byId))
	copy(out, r.byId)
	return out
}

func (r *Registry) Count() int { return len(r.byId) }
