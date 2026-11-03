// Package link holds the domain model and the contracts its consumers require.
// It may not import a transport or a storage package.
package link

import "time"

type Link struct {
	ID        uint64
	Code      string
	Target    string
	OwnerID   string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type ShortenRequest struct {
	Target    string
	Alias     string
	OwnerID   string
	ExpiresAt time.Time
}

type Page struct {
	Cursor string
	Limit  int
}
