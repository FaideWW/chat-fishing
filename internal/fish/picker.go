package fish

import (
	"crypto/rand"
	"encoding/binary"
	"math"
	mrand "math/rand"
	"time"
)

type Picker struct {
	reg         *Registry
	cumulative  []int
	totalWeight int
	meanWeight  float64
	rng         *mrand.Rand
}

func NewPicker(reg *Registry, rng *mrand.Rand) *Picker {
	if rng == nil {
		var b [8]byte
		if _, err := rand.Read(b[:]); err != nil {
			rng = mrand.New(mrand.NewSource(time.Now().UnixNano()))
		} else {
			rng = mrand.New(mrand.NewSource(int64(binary.LittleEndian.Uint64(b[:]))))
		}
	}

	p := &Picker{
		reg: reg,
		rng: rng,
	}

	all := reg.All()

	p.cumulative = make([]int, len(all))
	totalWeight := 0
	for i, sp := range all {
		if sp.Weight < 1 {
			sp.Weight = 1
		}
		totalWeight += sp.Weight
		p.cumulative[i] = totalWeight
	}
	p.totalWeight = totalWeight
	p.meanWeight = float64(totalWeight) / float64(len(all))
	return p
}

func (p *Picker) PickId() SpeciesId {
	roll := p.rng.Intn(p.totalWeight) // random int from [0,totalWeight)

	// binary search for the species using p.cumulative
	lo, hi := 0, len(p.cumulative)-1
	for lo < hi {
		mid := (lo + hi) >> 1
		if roll < p.cumulative[mid] {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	return SpeciesId(lo)
}

// Sizes are determined by u^k, where u is a random value between 0 and 1
// and k is the size bias of the species. A higher k means the fish tend
// to be smaller, where as k = 1 is a uniform distribution.
func (p *Picker) RollSize(id SpeciesId) float64 {
	sp, ok := p.reg.GetById(id)
	if !ok {
		return 0
	}
	min, max := sp.MinSize, sp.MaxSize
	if max < min {
		max = min
	}
	u := p.rng.Float64()
	k := sp.SizeBias
	if k < 1 {
		k = 1
	}
	scaled := math.Pow(u, k)
	size := min + (max-min)*scaled

	// sizes are rounded to the mm
	return math.Round(size*10) / 10
}
