package httpx

// Request DTOs. The json tags are the wire contract.

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
