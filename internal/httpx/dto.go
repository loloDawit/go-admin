package httpx

// Request DTOs.
//
// Every handler parses into one of these rather than map[string]string. The
// original code read request fields by string key, which meant the wire
// contract existed only as literals scattered through handlers — and two
// handlers disagreed about the same field: Register read "firstName" while
// UpdateUserInfo read "firstname". The frontend sends camelCase, so renaming
// yourself through the UI silently did nothing (ASSESSMENT 4x).
//
// With a struct, the json tag IS the contract, it is declared once, and the
// compiler checks every use of the field.

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"passwordConfirm"`
}

type UpdateUserInfoRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
}

type UpdatePasswordRequest struct {
	Password        string `json:"password"`
	PasswordConfirm string `json:"passwordConfirm"`
}

type CreateUserRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	RoleId    uint   `json:"roleId"`
}

type UpdateUserRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	RoleId    uint   `json:"roleId"`
}

// RoleRequest replaces the unchecked fiber.Map type assertions in
// role_controller.go, which panicked whenever a client sent permission ids as
// JSON numbers rather than strings (ASSESSMENT 4n).
type RoleRequest struct {
	Name        string `json:"name"`
	Permissions []uint `json:"permissions"`
}

type PermissionRequest struct {
	Name string `json:"name"`
}

type ProductRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Image       string  `json:"image"`
	Price       float64 `json:"price"`
}
