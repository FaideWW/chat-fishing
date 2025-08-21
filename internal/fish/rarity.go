package fish

type RarityTier int

const (
	TierCommon RarityTier = iota
	TierUncommon
	TierRare
	TierEpic
	TierLegendary
	TierMythic
)

func (t RarityTier) String() string {
	switch t {
	case TierMythic:
		return "Mythic"
	case TierLegendary:
		return "Legendary"
	case TierEpic:
		return "Epic"
	case TierRare:
		return "Rare"
	case TierUncommon:
		return "Uncommon"
	default:
		return "Common"
	}
}

func (p *Picker) SpeciesTier(id SpeciesId) RarityTier {
	sp, ok := p.reg.GetById(id)
	if !ok {
		return TierCommon
	}
	r := float64(sp.Weight) / p.meanWeight
	switch {
	case r < 0.05:
		return TierMythic
	case r < 0.20:
		return TierLegendary
	case r < 0.50:
		return TierEpic
	case r < 1.00:
		return TierRare
	case r < 1.50:
		return TierUncommon
	default:
		return TierCommon
	}
}

func ColorForTier(t RarityTier) int {
	switch t {
	case TierMythic:
		return 0xE74C3C // red
	case TierLegendary:
		return 0xF1C40F // gold
	case TierEpic:
		return 0x9B59B6 // purple
	case TierRare:
		return 0x3498DB // blue
	case TierUncommon:
		return 0x2ECC71 // green
	default:
		return 0x95A5A6 // gray
	}
}
