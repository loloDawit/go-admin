package staff

import "strconv"

// CreateStaffRequest is POST /api/v1/staff's body.
type CreateStaffRequest struct {
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	RoleID    string `json:"roleId"`
}

// StaffResponse is the shape every route answers a single staff record with; roleId is a string, matching AuthResponse's staffId.
type StaffResponse struct {
	ID                 string `json:"id"`
	Email              string `json:"email"`
	FirstName          string `json:"firstName"`
	LastName           string `json:"lastName"`
	RoleID             string `json:"roleId"`
	IsActive           bool   `json:"isActive"`
	MustChangePassword bool   `json:"mustChangePassword"`
}

// CreateStaffResponse carries the generated password once, alongside the
// created record. Nothing else in this package ever answers with it again.
type CreateStaffResponse struct {
	Staff    StaffResponse `json:"staff"`
	Password string        `json:"password"`
}

// StaffPageResponse matches the shape every other paged list in the system
// answers with: items plus the effective page, size and total.
type StaffPageResponse struct {
	Items    []StaffResponse `json:"items"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
	Total    int             `json:"total"`
}

// AdminUpdateStaffRequest is PATCH /api/v1/staff/{id}'s body for updating someone other than the caller; every field is optional and a nil pointer leaves that column unchanged.
type AdminUpdateStaffRequest struct {
	FirstName *string `json:"firstName"`
	LastName  *string `json:"lastName"`
	Email     *string `json:"email"`
	RoleID    *string `json:"roleId"`
	IsActive  *bool   `json:"isActive"`
}

type SelfUpdateStaffRequest struct {
	FirstName *string `json:"firstName"`
	LastName  *string `json:"lastName"`
	Email     *string `json:"email"`
}

// ChangePasswordRequest is POST /api/v1/me/password's body.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func newStaffResponse(st Staff) StaffResponse {
	return StaffResponse{
		ID:                 strconv.FormatInt(st.ID, 10),
		Email:              st.Email,
		FirstName:          st.FirstName,
		LastName:           st.LastName,
		RoleID:             strconv.FormatInt(st.RoleID, 10),
		IsActive:           st.IsActive,
		MustChangePassword: st.MustChangePassword,
	}
}

func newStaffPageResponse(p Page) StaffPageResponse {
	out := make([]StaffResponse, len(p.Items))
	for i, st := range p.Items {
		out[i] = newStaffResponse(st)
	}
	return StaffPageResponse{Items: out, Page: p.Page, PageSize: p.PageSize, Total: p.Total}
}
