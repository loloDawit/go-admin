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
	MemberCount int      `json:"memberCount"`
}

// RolePageResponse matches the shape every other paged list answers with.
type RolePageResponse struct {
	Items    []RoleResponse `json:"items"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
	Total    int            `json:"total"`
}

func newRoleResponse(r Role) RoleResponse {
	return RoleResponse{
		ID:          strconv.FormatInt(r.ID, 10),
		Name:        r.Name,
		Permissions: r.Permissions,
		MemberCount: r.MemberCount,
	}
}

func newRolePageResponse(p Page) RolePageResponse {
	out := make([]RoleResponse, len(p.Items))
	for i, r := range p.Items {
		out[i] = newRoleResponse(r)
	}
	return RolePageResponse{Items: out, Page: p.Page, PageSize: p.PageSize, Total: p.Total}
}
