package domain

import "time"

type Group struct {
	ID        int64
	Name      string
	UserID    int64
	CreatedAt time.Time
}
