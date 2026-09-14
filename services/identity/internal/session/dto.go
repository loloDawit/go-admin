package session

import "strconv"

// LoginRequest is POST /api/v1/login's body.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse is POST /api/v1/login's body. staffId is a string: a BIGSERIAL can exceed JavaScript's safe integer range.
type AuthResponse struct {
	StaffID            string   `json:"staffId"`
	Email              string   `json:"email"`
	Permissions        []string `json:"permissions"`
	MustChangePassword bool     `json:"mustChangePassword"`
}

func newAuthResponse(a Authenticated) AuthResponse {
	return AuthResponse{
		StaffID:            strconv.FormatInt(a.StaffID, 10),
		Email:              a.Email,
		Permissions:        a.Permissions,
		MustChangePassword: a.MustChangePassword,
	}
}

// GET /api/v1/me shares AuthResponse's shape with login: both answer with
// the caller's current record, email included.

// ValidateRequest carries the token in JSON, never a query string: a query string reaches access logs.
type ValidateRequest struct {
	Token string `json:"token"`
}

// ValidateResponse carries no issuedAt/expiresAt: those are the caller's own to set against its own TTL.
type ValidateResponse struct {
	StaffID            string   `json:"staffId"`
	Permissions        []string `json:"permissions"`
	MustChangePassword bool     `json:"mustChangePassword"`
}
