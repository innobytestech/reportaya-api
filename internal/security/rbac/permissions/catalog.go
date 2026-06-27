package permissions

import "sort"

// Code is a stable RBAC permission code (e.g. "users.read"). Domains declare
// their own codes here as they are built; the authorizer only ever compares the
// codes a user has against the code a route requires.
type Code string

// Permission is a registered permission with human-readable metadata, used to
// seed/expose the catalog (e.g. an admin UI listing assignable permissions).
type Permission struct {
	Code        Code
	Description string
	Module      string
}

// Skeleton baseline permissions. Extend per domain as modules are added.
const (
	UsersRead   Code = "users.read"
	UsersCreate Code = "users.create"
	UsersUpdate Code = "users.update"
	UsersDelete Code = "users.delete"

	RBACRead   Code = "rbac.read"
	RBACAssign Code = "rbac.assign"
	RBACRevoke Code = "rbac.revoke"

	AuditRead Code = "audit.read"
)

// catalog holds every known permission keyed by code.
var catalog = map[Code]Permission{
	UsersRead:   {Code: UsersRead, Description: "Read users", Module: "users"},
	UsersCreate: {Code: UsersCreate, Description: "Create users", Module: "users"},
	UsersUpdate: {Code: UsersUpdate, Description: "Update users", Module: "users"},
	UsersDelete: {Code: UsersDelete, Description: "Delete users", Module: "users"},
	RBACRead:    {Code: RBACRead, Description: "Read roles and permissions", Module: "rbac"},
	RBACAssign:  {Code: RBACAssign, Description: "Assign roles/permissions", Module: "rbac"},
	RBACRevoke:  {Code: RBACRevoke, Description: "Revoke roles/permissions", Module: "rbac"},
	AuditRead:   {Code: AuditRead, Description: "Read audit log", Module: "audit"},
}

// All returns every registered permission, sorted by code (stable output).
func All() []Permission {
	out := make([]Permission, 0, len(catalog))
	for _, p := range catalog {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}

// IsValid reports whether code is a registered permission.
func IsValid(code Code) bool {
	_, ok := catalog[code]
	return ok
}
