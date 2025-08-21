package ratelimit

import (
	"crypto/rand"
	"encoding/binary"
	mrand "math/rand"
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
}

type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }

type Limiter struct {
	mu   sync.Mutex
	next map[string]time.Time
	min  time.Duration
	max  time.Duration
	clk  Clock
	rng  *mrand.Rand
}

func NewLimiter(min, max time.Duration, clk Clock) *Limiter {
	if clk == nil {
		clk = RealClock{}
	}
	if max < min {
		max = min
	}

	seed := func() int64 {
		var b [8]byte
		if _, err := rand.Read(b[:]); err == nil {
			return int64(binary.LittleEndian.Uint64(b[:]))
		}
		return time.Now().UnixNano()
	}()

	return &Limiter{
		next: make(map[string]time.Time),
		min:  min,
		max:  max,
		clk:  clk,
		rng:  mrand.New(mrand.NewSource(seed)),
	}
}

func (l *Limiter) TryKey(key string) (bool, time.Duration) {
	now := l.clk.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	if until, ok := l.next[key]; ok && now.Before(until) {
		return false, time.Until(until)
	}

	l.next[key] = now.Add(l.nextCooldown())
	return true, 0
}

func (l *Limiter) Try(guildId, userId string) (bool, time.Duration) {
	return l.TryKey(guildId + ":" + userId)
}

func (l *Limiter) TryGuild(guildId, bucket string) (bool, time.Duration) {
	return l.TryKey("g:" + guildId + "|b:" + bucket)
}

func (l *Limiter) nextCooldown() time.Duration {
	if l.min == l.max {
		return l.min
	}
	span := l.max - l.min

	jitter := time.Duration(l.rng.Int63n(int64(span)))
	return l.min + jitter
}

func (l *Limiter) Reset(guildId, userId string) {
	l.mu.Lock()
	delete(l.next, guildId+":"+userId)
	l.mu.Unlock()
}

func (l *Limiter) Peek(guildId, userId string) (time.Time, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	t, ok := l.next[guildId+":"+userId]
	return t, ok
}
