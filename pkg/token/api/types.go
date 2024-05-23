/*
   Copyright 2024 The olive Authors

   This library is free software; you can redistribute it and/or
   modify it under the terms of the GNU Lesser General Public
   License as published by the Free Software Foundation; either
   version 2.1 of the License, or (at your option) any later version.

   This library is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
   Lesser General Public License for more details.

   You should have received a copy of the GNU Lesser General Public
   License along with this library;
*/

package api

const (
	// BootstrapGroupPattern is the valid regex pattern that all groups
	// assigned to a bootstrap token by BootstrapTokenExtraGroupsKey must match.
	// See also util.ValidateBootstrapGroupName()
	BootstrapGroupPattern = `\Asystem:bootstrappers:[a-z0-9:-]{0,255}[a-z0-9]\z`

	// BootstrapTokenSecretPrefix is the prefix for bootstrap token names.
	// Bootstrap tokens secrets must be named in the form
	// `bootstrap-token-<token-id>`.  This is the prefix to be used before the
	// token ID.
	BootstrapTokenSecretPrefix = "bootstrap-token-"

	// BootstrapTokenPattern defines the {id}.{secret} regular expression pattern
	BootstrapTokenPattern = `\A([a-z0-9]{6})\.([a-z0-9]{16})\z`

	// BootstrapTokenIDPattern defines token's id regular expression pattern
	BootstrapTokenIDPattern = `\A([a-z0-9]{6})\z`

	// BootstrapTokenIDBytes defines the number of bytes used for the Bootstrap Token's ID field
	BootstrapTokenIDBytes = 6

	// BootstrapTokenSecretBytes defines the number of bytes used the Bootstrap Token's Secret field
	BootstrapTokenSecretBytes = 16
)

// KnownTokenUsages specifies the known functions a token will get.
var KnownTokenUsages = []string{"signing", "authentication"}
