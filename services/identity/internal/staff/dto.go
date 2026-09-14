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

type ListStaffResponse struct {
	Staff []StaffResponse `json:"staff"`
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

func newListStaffResponse(list []Staff) ListStaffResponse {
	out := make([]StaffResponse, len(list))
	for i, st := range list {
		out[i] = newStaffResponse(st)
	}
	return ListStaffResponse{Staff: out}
}
