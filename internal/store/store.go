package store

import (
	"context"

	"github.com/faideww/chat-fishing/internal/fish"
)

type Store interface {
	Add(ctx context.Context, c fish.Catch) error
	AddBatch(ctx context.Context, cs []fish.Catch) error
	TopBySize(ctx context.Context, guildId uint64, limit int) ([]fish.Catch, error)
}
