// Package principal holds signing only: the permission vocabulary is Identity's published contract, not shared platform code.
package principal

import "time"

type Principal struct {
	StaffID            string
	Permissions        []string
	MustChangePassword bool // lets a downstream service enforce a forced password change without a database read
	IssuedAt           time.Time
	ExpiresAt          time.Time
}
