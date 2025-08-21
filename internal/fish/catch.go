package fish

import "time"

// Catch is the canonical record used by handlers and stores
// Storage backends should persist size_tenths (int) for precision/ordering
type Catch struct {
	Id        int64
	GuildId   int64
	UserId    int64
	SpeciesId SpeciesId
	Species   string
	Size      float64
	CaughtAt  time.Time
}
