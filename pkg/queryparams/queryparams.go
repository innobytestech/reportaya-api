package queryparams

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// String returns the first non-empty query value among keys.
func String(c *fiber.Ctx, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(c.Query(k)); v != "" {
			return v
		}
	}
	return ""
}

// BoolPtr parses an optional bool query param.
func BoolPtr(c *fiber.Ctx, keys ...string) (*bool, error) {
	raw := String(c, keys...)
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid boolean value %q", raw)
	}
	return &v, nil
}

// UUIDPtr parses an optional UUID query param.
func UUIDPtr(c *fiber.Ctx, keys ...string) (*uuid.UUID, error) {
	raw := String(c, keys...)
	if raw == "" {
		return nil, nil
	}
	v, err := uuid.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid uuid value %q", raw)
	}
	return &v, nil
}

// UUID parses a required UUID query param.
func UUID(c *fiber.Ctx, key string) (uuid.UUID, error) {
	raw := String(c, key)
	if raw == "" {
		return uuid.Nil, fmt.Errorf("%s is required", key)
	}
	v, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s", key)
	}
	return v, nil
}

// UUIDList parses an optional repeatable/CSV UUID query param. It accepts both
// `?key=a,b` and `?key=a&key=b` (and mixes of the two), de-duplicating while
// preserving first-seen order. Returns an error on any malformed UUID.
func UUIDList(c *fiber.Ctx, key string) ([]uuid.UUID, error) {
	raw := c.Context().QueryArgs().PeekMulti(key)
	seen := make(map[uuid.UUID]struct{})
	out := make([]uuid.UUID, 0, len(raw))
	for _, b := range raw {
		for _, part := range strings.Split(string(b), ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := uuid.Parse(part)
			if err != nil {
				return nil, fmt.Errorf("invalid uuid value %q", part)
			}
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out, nil
}

// TimePtrRFC3339 parses an optional RFC3339 timestamp query param.
func TimePtrRFC3339(c *fiber.Ctx, key string) (*time.Time, error) {
	raw := String(c, key)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, fmt.Errorf("invalid %s, expected RFC3339", key)
	}
	return &t, nil
}

// Search returns normalized free-text search term from search or q.
func Search(c *fiber.Ctx) string {
	return String(c, "search", "q")
}

// ParseSort maps requested sortBy/order to safe DB column and direction.
func ParseSort(c *fiber.Ctx, allowed map[string]string, defaultKey, defaultDir string) (string, string, error) {
	if len(allowed) == 0 {
		return "", "", fmt.Errorf("allowed sort map is required")
	}

	requestedKey := strings.TrimSpace(c.Query("sortBy", defaultKey))
	if requestedKey == "" {
		requestedKey = defaultKey
	}

	dir := strings.ToUpper(strings.TrimSpace(c.Query("order", c.Query("sortOrder", defaultDir))))
	if dir == "" {
		dir = strings.ToUpper(defaultDir)
	}
	if dir != "ASC" && dir != "DESC" {
		return "", "", fmt.Errorf("invalid order, allowed values: ASC or DESC")
	}

	column, ok := allowed[requestedKey]
	if !ok {
		keys := make([]string, 0, len(allowed))
		for k := range allowed {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return "", "", fmt.Errorf("invalid sortBy %q, allowed values: %s", requestedKey, strings.Join(keys, ", "))
	}

	return column, dir, nil
}
