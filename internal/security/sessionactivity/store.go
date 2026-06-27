// Package sessionactivity tracks the timestamp of the last authenticated
// request per user so the AuthService can enforce an idle session timeout.
//
// The store is intentionally narrow: it only supports a throttled write
// (Touch) and a lookup (LastActivity). The HTTP middleware writes via Touch
// on every authenticated request; AuthService.Refresh reads via LastActivity
// to decide whether to reject the rotation as idle-expired.
package sessionactivity

import (
	"context"
	"fmt"
	"time"
)

// Store persists the last activity time per user.
type Store interface {
	// Touch records that the user was active "now". The throttle controls how
	// often the underlying write actually fires per user; calls inside the
	// window are dropped silently. activityTTL is the TTL of the activity
	// record itself and should be > the configured idle timeout so that a
	// LastActivity lookup can distinguish "expired" from "never set".
	Touch(ctx context.Context, userID string, activityTTL time.Duration, throttle time.Duration) error
	// LastActivity returns the last recorded activity time for the user.
	// The bool is false when no record exists (either the user has never
	// been active or the record TTL elapsed); in either case the caller
	// should treat the session as idle-expired.
	LastActivity(ctx context.Context, userID string) (time.Time, bool, error)
}

// BuildActivityKey returns the Redis key for a user's last-activity timestamp.
func BuildActivityKey(userID string) string {
	return fmt.Sprintf("session:activity:%s", userID)
}

// BuildThrottleKey returns the Redis key for the per-user write throttle gate.
func BuildThrottleKey(userID string) string {
	return fmt.Sprintf("session:activity:throttle:%s", userID)
}
