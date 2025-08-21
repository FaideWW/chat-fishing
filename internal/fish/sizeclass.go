package fish

import "math"

type SizeClass int

const (
	SizeTiny SizeClass = iota
	SizeSmall
	SizeAverage
	SizeBig
	SizeHuge
	SizeEnormous
)

func (c SizeClass) String() string {
	switch c {
	case SizeTiny:
		return "tiny"
	case SizeSmall:
		return "modest"
	case SizeAverage:
		return "average"
	case SizeBig:
		return "big"
	case SizeHuge:
		return "huge"
	default:
		return "enormous"
	}
}

func SizePercentile(sp Species, size float64) float64 {
	if sp.MaxSize <= sp.MinSize {
		return 0
	}

	x := (size - sp.MinSize) / (sp.MaxSize - sp.MinSize)
	if x < 0 {
		x = 0
	} else if x > 1 {
		x = 1
	}
	k := sp.SizeBias
	if k <= 0 {
		k = 1
	}
	return math.Pow(x, 1.0/k) // CDF
}

func ClassFromPercentile(p float64) SizeClass {
	switch {
	case p < 0.08:
		return SizeTiny
	case p < 0.25:
		return SizeSmall
	case p < 0.70:
		return SizeAverage
	case p < 0.90:
		return SizeBig
	case p < 0.97:
		return SizeHuge
	default:
		return SizeEnormous
	}
}

func SizeClassFor(sp Species, size float64) SizeClass {
	return ClassFromPercentile(SizePercentile(sp, size))
}
