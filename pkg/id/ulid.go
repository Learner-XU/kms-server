package id

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// New generates a new ULID
func New() string {
	return ulid.Make().String()
}

// NewWithTime generates a ULID with a specific timestamp
func NewWithTime(t time.Time) string {
	return ulid.MustNew(ulid.Timestamp(t), ulid.Monotonic(rand.Reader, 0)).String()
}
