package rbac

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"reportaya-api/internal/domain/rbac/models"
	usermodels "reportaya-api/internal/domain/user/models"
	rbacperm "reportaya-api/internal/security/rbac/permissions"
)

const (
	// SuperAdminRoleName is the staff role that bypasses permission checks.
	SuperAdminRoleName    = "admin"
	rbacInvalidateChannel = "rbac:invalidate"
	rbacPermKeyPrefix     = "rbac:perms:user:"
	rbacPermUsersSetKey   = "rbac:perms:users"
)

// Authorizer loads and caches user permissions and checks access.
type Authorizer struct {
	db       *gorm.DB
	redis    *redis.Client
	cache    map[uuid.UUID]*userPerms
	cacheTTL time.Duration
	mu       sync.RWMutex

	subCtx    context.Context
	subCancel context.CancelFunc
}

type userPerms struct {
	perms   map[string]bool
	isAdmin bool
	expires time.Time
}

type redisUserPerms struct {
	Perms   map[string]bool `json:"perms"`
	IsAdmin bool            `json:"is_admin"`
}

type invalidateMessage struct {
	Type   string `json:"type"`
	UserID string `json:"user_id,omitempty"`
}

// NewAuthorizer creates an RBAC authorizer with in-memory TTL cache.
func NewAuthorizer(db *gorm.DB, redisClient *redis.Client, cacheTTL time.Duration) *Authorizer {
	ctx, cancel := context.WithCancel(context.Background())
	a := &Authorizer{
		db:        db,
		redis:     redisClient,
		cacheTTL:  cacheTTL,
		cache:     make(map[uuid.UUID]*userPerms),
		subCtx:    ctx,
		subCancel: cancel,
	}
	if redisClient != nil {
		go a.runInvalidationSubscriber(ctx)
	}
	return a
}

// Stop cancels the background invalidation subscriber goroutine. Safe to call
// multiple times. Intended to be wired to the application lifecycle shutdown.
func (a *Authorizer) Stop() {
	if a.subCancel != nil {
		a.subCancel()
	}
}

// Can returns true if the user has the permission or is staff admin.
func (a *Authorizer) Can(ctx context.Context, userID uuid.UUID, realm usermodels.UserRealm, permission rbacperm.Code) (bool, error) {
	p, err := a.getUserPerms(ctx, userID, realm)
	if err != nil {
		return false, err
	}
	if p.isAdmin && realm == usermodels.RealmStaff {
		return true, nil
	}
	return p.perms[string(permission)], nil
}

// PermissionCodes returns cached/evaluated permission codes for a user.
func (a *Authorizer) PermissionCodes(ctx context.Context, userID uuid.UUID, realm usermodels.UserRealm) ([]string, bool, error) {
	p, err := a.getUserPerms(ctx, userID, realm)
	if err != nil {
		return nil, false, err
	}
	codes := make([]string, 0, len(p.perms))
	for code := range p.perms {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes, p.isAdmin && realm == usermodels.RealmStaff, nil
}

func (a *Authorizer) getUserPerms(ctx context.Context, userID uuid.UUID, realm usermodels.UserRealm) (*userPerms, error) {
	a.mu.RLock()
	if c, ok := a.cache[userID]; ok && time.Now().Before(c.expires) {
		a.mu.RUnlock()
		return c, nil
	}
	a.mu.RUnlock()

	if a.redis != nil {
		if p, ok, err := a.getUserPermsFromRedis(ctx, userID); err == nil && ok {
			a.mu.Lock()
			a.cache[userID] = p
			a.mu.Unlock()
			return p, nil
		}
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	// Double-check after lock
	if c, ok := a.cache[userID]; ok && time.Now().Before(c.expires) {
		return c, nil
	}

	// Load from DB: user -> user_roles -> roles -> role_permissions -> permissions
	var roleIDs []uuid.UUID
	if err := a.db.WithContext(ctx).Table("user_roles").Where("user_id = ?", userID).Pluck("role_id", &roleIDs).Error; err != nil {
		return nil, err
	}
	if len(roleIDs) == 0 {
		a.cache[userID] = &userPerms{perms: map[string]bool{}, expires: time.Now().Add(a.cacheTTL)}
		a.cacheUserPermsToRedis(ctx, userID, a.cache[userID])
		return a.cache[userID], nil
	}

	var roles []models.Role
	if err := a.db.WithContext(ctx).Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
		return nil, err
	}
	activeRoleIDs := make([]uuid.UUID, 0, len(roles))
	isAdmin := false
	for _, r := range roles {
		activeRoleIDs = append(activeRoleIDs, r.ID)
		if r.Realm == string(usermodels.RealmStaff) && r.Name == SuperAdminRoleName {
			isAdmin = true
			break
		}
	}
	if len(activeRoleIDs) == 0 {
		a.cache[userID] = &userPerms{perms: map[string]bool{}, isAdmin: false, expires: time.Now().Add(a.cacheTTL)}
		a.cacheUserPermsToRedis(ctx, userID, a.cache[userID])
		return a.cache[userID], nil
	}

	var permissionIDs []uuid.UUID
	if err := a.db.WithContext(ctx).Table("role_permissions").Where("role_id IN ?", activeRoleIDs).Pluck("permission_id", &permissionIDs).Error; err != nil {
		return nil, err
	}
	if len(permissionIDs) == 0 && !isAdmin {
		a.cache[userID] = &userPerms{perms: map[string]bool{}, isAdmin: isAdmin, expires: time.Now().Add(a.cacheTTL)}
		a.cacheUserPermsToRedis(ctx, userID, a.cache[userID])
		return a.cache[userID], nil
	}

	var perms []models.Permission
	if err := a.db.WithContext(ctx).Where("id IN ?", permissionIDs).Find(&perms).Error; err != nil {
		return nil, err
	}
	pm := make(map[string]bool)
	for _, p := range perms {
		pm[p.Code] = true
	}
	a.cache[userID] = &userPerms{perms: pm, isAdmin: isAdmin, expires: time.Now().Add(a.cacheTTL)}
	a.cacheUserPermsToRedis(ctx, userID, a.cache[userID])
	return a.cache[userID], nil
}

// InvalidateUser clears cache for a user (call after role/permission changes).
func (a *Authorizer) InvalidateUser(userID uuid.UUID) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.cache, userID)
	a.invalidateUserRedis(context.Background(), userID)
	a.publishInvalidation(invalidateMessage{Type: "user", UserID: userID.String()})
}

// InvalidateAll clears the full permission cache.
func (a *Authorizer) InvalidateAll() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cache = make(map[uuid.UUID]*userPerms)
	a.invalidateAllRedis(context.Background())
	a.publishInvalidation(invalidateMessage{Type: "all"})
}

func (a *Authorizer) redisUserPermsKey(userID uuid.UUID) string {
	return rbacPermKeyPrefix + userID.String()
}

func (a *Authorizer) getUserPermsFromRedis(ctx context.Context, userID uuid.UUID) (*userPerms, bool, error) {
	s, err := a.redis.Get(ctx, a.redisUserPermsKey(userID)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var payload redisUserPerms
	if err := json.Unmarshal([]byte(s), &payload); err != nil {
		return nil, false, err
	}
	if payload.Perms == nil {
		payload.Perms = map[string]bool{}
	}
	return &userPerms{
		perms:   payload.Perms,
		isAdmin: payload.IsAdmin,
		expires: time.Now().Add(a.cacheTTL),
	}, true, nil
}

func (a *Authorizer) cacheUserPermsToRedis(ctx context.Context, userID uuid.UUID, perms *userPerms) {
	if a.redis == nil || perms == nil {
		return
	}
	payload := redisUserPerms{Perms: perms.perms, IsAdmin: perms.isAdmin}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	pipe := a.redis.Pipeline()
	pipe.Set(ctx, a.redisUserPermsKey(userID), string(b), a.cacheTTL)
	pipe.SAdd(ctx, rbacPermUsersSetKey, userID.String())
	_, _ = pipe.Exec(ctx)
}

func (a *Authorizer) invalidateUserRedis(ctx context.Context, userID uuid.UUID) {
	if a.redis == nil {
		return
	}
	pipe := a.redis.Pipeline()
	pipe.Del(ctx, a.redisUserPermsKey(userID))
	pipe.SRem(ctx, rbacPermUsersSetKey, userID.String())
	_, _ = pipe.Exec(ctx)
}

func (a *Authorizer) invalidateAllRedis(ctx context.Context) {
	if a.redis == nil {
		return
	}
	userIDs, err := a.redis.SMembers(ctx, rbacPermUsersSetKey).Result()
	if err != nil {
		return
	}
	if len(userIDs) == 0 {
		_ = a.redis.Del(ctx, rbacPermUsersSetKey).Err()
		return
	}
	keys := make([]string, 0, len(userIDs)+1)
	for _, id := range userIDs {
		keys = append(keys, rbacPermKeyPrefix+id)
	}
	keys = append(keys, rbacPermUsersSetKey)
	_ = a.redis.Del(ctx, keys...).Err()
}

func (a *Authorizer) publishInvalidation(msg invalidateMessage) {
	if a.redis == nil {
		return
	}
	b, err := json.Marshal(msg)
	if err != nil {
		return
	}
	_ = a.redis.Publish(context.Background(), rbacInvalidateChannel, string(b)).Err()
}

func (a *Authorizer) runInvalidationSubscriber(ctx context.Context) {
	if a.redis == nil {
		return
	}
	pubsub := a.redis.Subscribe(ctx, rbacInvalidateChannel)
	defer func() { _ = pubsub.Close() }()

	// ReceiveMessage on a healthy connection blocks regardless of ctx, so close
	// the subscription on cancellation to force it to return; the ctx.Err()
	// check below then ends the loop instead of retrying forever.
	go func() {
		<-ctx.Done()
		_ = pubsub.Close()
	}()

	for {
		// Stop when the app lifecycle ctx is cancelled; otherwise a Redis
		// outage would spin this loop forever via the Sleep+continue path.
		select {
		case <-ctx.Done():
			return
		default:
		}
		msg, err := pubsub.ReceiveMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			// Backoff on transient errors, but stay cancellable.
			select {
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
			}
			continue
		}
		var evt invalidateMessage
		if err := json.Unmarshal([]byte(msg.Payload), &evt); err != nil {
			continue
		}
		switch evt.Type {
		case "all":
			a.mu.Lock()
			a.cache = make(map[uuid.UUID]*userPerms)
			a.mu.Unlock()
		case "user":
			if evt.UserID == "" {
				continue
			}
			uid, err := uuid.Parse(evt.UserID)
			if err != nil {
				continue
			}
			a.mu.Lock()
			delete(a.cache, uid)
			a.mu.Unlock()
		}
	}
}
