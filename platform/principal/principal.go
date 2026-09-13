// Package principal signs and verifies the staff identity a gateway attaches
// to every request. It holds signing only: the permission vocabulary is
// Identity's published contract, not shared platform code, so this package
// never declares what a valid permission string is.
package principal

import "time"

type Principal struct {
	StaffID     string
	Permissions []string
	// MustChangePassword travels on the signed principal so a downstream
	// service can enforce a forced password change without a database read
	// on the request path.
	MustChangePassword bool
	IssuedAt           time.Time
	ExpiresAt          time.Time
}
