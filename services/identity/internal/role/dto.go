package role

import "strconv"

// CreateRoleRequest is POST /api/v1/roles's body.
type CreateRoleRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

// UpdateRoleRequest is PATCH /api/v1/roles/{id}'s body; every field is
// optional and a nil pointer leaves that column unchanged.
type UpdateRoleRequest struct {
	Name        *string   `json:"name"`
	Permissions *[]string `json:"permissions"`
}

// RoleResponse is the shape every route answers a single role with; id is a
// string, matching StaffResponse's roleId.
type RoleResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

type ListRoleResponse struct {
	Roles []RoleResponse `json:"roles"`
}

func newRoleResponse(r Role) RoleResponse {
	return RoleResponse{ID: strconv.FormatInt(r.ID, 10), Name: r.Name, Permissions: r.Permissions}
}

func newListRoleResponse(list []Role) ListRoleResponse {
	out := make([]RoleResponse, len(list))
	for i, r := range list {
		out[i] = newRoleResponse(r)
	}
	return ListRoleResponse{Roles: out}
}
