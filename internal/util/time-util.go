package util

import (
	"time"
)

// IsOutdated returns true if the timestamp is less than the timestamp + lifetime
func IsOutdated(timestamp time.Time, lifetime time.Duration) bool {
	elapsed := time.Now().Sub(timestamp)
	return elapsed >= lifetime
}

// IsExpired returns true if the timestamp is before the current datetime
func IsExpired(timestamp time.Time) bool {
	return time.Now().After(timestamp)
}
