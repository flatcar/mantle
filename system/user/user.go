// Copyright The Mantle Authors
// SPDX-License-Identifier: Apache-2.0

// user provides an extended User struct to simplify usage
package user

import (
	"os/user"
	"strconv"
)

// User represents a user account.
type User struct {
	*user.User
	Groupname string
	// For convenience so users don't need to strconv themselves.
	UidNo int
	GidNo int
}

// Current returns the current user.
func Current() (*User, error) {
	return newUser(user.Current())
}

// Lookup looks up a user by username.
func Lookup(username string) (*User, error) {
	return newUser(user.Lookup(username))
}

// LookupId looks up a user by userid.
func LookupId(uid string) (*User, error) {
	return newUser(user.LookupId(uid))
}

// Convert the stock user.User to our own User strict with group info.
func newUser(u *user.User, err error) (*User, error) {
	if err != nil {
		return nil, err
	}

	g, err := user.LookupGroupId(u.Gid)
	if err != nil {
		return nil, err
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return nil, err
	}

	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return nil, err
	}

	return &User{
		User:      u,
		Groupname: g.Name,
		UidNo:     uid,
		GidNo:     gid,
	}, nil
}
