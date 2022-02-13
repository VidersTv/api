package modelstructures

import (
	"time"

	"github.com/viderstv/api/graph/model"
	"github.com/viderstv/common/structures"
)

type Stream structures.Stream

func (s Stream) ToModel() *model.Stream {
	var endedAt *time.Time
	if !s.EndedAt.IsZero() {
		endedAt = &s.EndedAt
	}

	return &model.Stream{
		ID:        s.ID,
		UserID:    s.UserID,
		Title:     s.Title,
		StartedAt: s.StartedAt,
		EndedAt:   endedAt,
	}
}
