package session

// LoginRequest is POST /api/v1/login's body. DisallowUnknownFields (via
// httpx.DecodeJSON) rejects any other field outright, rather than silently
// ignoring it.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse is POST /api/v1/login's body: the full Authenticated record
// Login returns, including email.
type AuthResponse struct {
	StaffID            int64    `json:"staffId"`
	Email              string   `json:"email"`
	Permissions        []string `json:"permissions"`
	MustChangePassword bool     `json:"mustChangePassword"`
}

func newAuthResponse(a Authenticated) AuthResponse {
	return AuthResponse{
		StaffID:            a.StaffID,
		Email:              a.Email,
		Permissions:        a.Permissions,
		MustChangePassword: a.MustChangePassword,
	}
}

// GET /api/v1/me shares AuthResponse's shape with login: both answer with
// the caller's current record, email included.

// ValidateRequest is POST /internal/sessions/validate's body. The token
// travels here, in JSON, never as a query string: a query string reaches
// access logs.
type ValidateRequest struct {
	Token string `json:"token"`
}

// ValidateResponse is what the gateway (or any internal caller) uses to build
// its own signed principal; issuedAt/expiresAt are the caller's own to set
// against its own TTL, not identity's.
type ValidateResponse struct {
	StaffID            string   `json:"staffId"`
	Permissions        []string `json:"permissions"`
	MustChangePassword bool     `json:"mustChangePassword"`
}
