// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package user

// Login2 Object to receive user credentials with hash in JSON format
type Login2 struct {
	// The username used to log in.
	Username string `json:"username"`
	// The hash of the password for the user.
	Hash string `json:"hash"`
	// If true, the token returned will be valid a lot longer than default. Useful for "remember me" style logins.
	LongToken bool `json:"long_token"`
}
