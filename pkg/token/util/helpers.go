/*
Copyright 2024 The olive Authors

This program is offered under a commercial and under the AGPL license.
For AGPL licensing, see below.

AGPL licensing:
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package util

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/olive-io/olive/pkg/token/api"
)

// validBootstrapTokenChars defines the characters a bootstrap token can consist of
const validBootstrapTokenChars = "0123456789abcdefghijklmnopqrstuvwxyz"

var (
	// BootstrapTokenRegexp is a compiled regular expression of TokenRegexpString
	BootstrapTokenRegexp = regexp.MustCompile(api.BootstrapTokenPattern)
	// BootstrapTokenIDRegexp is a compiled regular expression of TokenIDRegexpString
	BootstrapTokenIDRegexp = regexp.MustCompile(api.BootstrapTokenIDPattern)
	// BootstrapGroupRegexp is a compiled regular expression of BootstrapGroupPattern
	BootstrapGroupRegexp = regexp.MustCompile(api.BootstrapGroupPattern)
)

// GenerateToken generates a new, random Token.
func GenerateToken() (string, error) {
	tokenID, err := RandBytes(api.BootstrapTokenIDBytes)
	if err != nil {
		return "", err
	}

	tokenSecret, err := RandBytes(api.BootstrapTokenSecretBytes)
	if err != nil {
		return "", err
	}

	return TokenFromIDAndSecret(tokenID, tokenSecret), nil
}

// RandBytes returns a random string consisting of the characters in
// validBootstrapTokenChars, with the length customized by the parameter
func RandBytes(length int) (string, error) {
	var (
		token = make([]byte, length)
		max   = new(big.Int).SetUint64(uint64(len(validBootstrapTokenChars)))
	)

	for i := range token {
		val, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("could not generate random integer: %w", err)
		}
		// Use simple operations in constant-time to obtain a byte in the a-z,0-9
		// character range
		x := val.Uint64()
		res := x + 48 + (39 & ((9 - x) >> 8))
		token[i] = byte(res)
	}

	return string(token), nil
}

// TokenFromIDAndSecret returns the full token which is of the form "{id}.{secret}"
func TokenFromIDAndSecret(id, secret string) string {
	return fmt.Sprintf("%s.%s", id, secret)
}

// IsValidBootstrapToken returns whether the given string is valid as a Bootstrap Token.
// Avoid using BootstrapTokenRegexp.MatchString(token) and instead perform constant-time
// comparisons on the secret.
func IsValidBootstrapToken(token string) bool {
	// Must be exactly two strings separated by "."
	t := strings.Split(token, ".")
	if len(t) != 2 {
		return false
	}

	// Validate the ID: t[0]
	// Using a Regexp for it is safe because the ID is public already
	if !BootstrapTokenIDRegexp.MatchString(t[0]) {
		return false
	}

	// Validate the secret with constant-time: t[1]
	secret := t[1]
	if len(secret) != api.BootstrapTokenSecretBytes { // Must be an exact size
		return false
	}
	for i := range secret {
		c := int(secret[i])
		notDigit := (c < 48 || c > 57)   // Character is not in the 0-9 range
		notLetter := (c < 97 || c > 122) // Character is not in the a-z range
		if notDigit && notLetter {
			return false
		}
	}
	return true
}

// IsValidBootstrapTokenID returns whether the given string is valid as a Bootstrap Token ID and
// in other words satisfies the BootstrapTokenIDRegexp
func IsValidBootstrapTokenID(tokenID string) bool {
	return BootstrapTokenIDRegexp.MatchString(tokenID)
}

// BootstrapTokenSecretName returns the expected name for the Secret storing the
// Bootstrap Token in the Kubernetes API.
func BootstrapTokenSecretName(tokenID string) string {
	return fmt.Sprintf("%s%s", api.BootstrapTokenSecretPrefix, tokenID)
}

// ValidateBootstrapGroupName checks if the provided group name is a valid
// bootstrap group name. Returns nil if valid or a validation error if invalid.
func ValidateBootstrapGroupName(name string) error {
	if BootstrapGroupRegexp.Match([]byte(name)) {
		return nil
	}
	return fmt.Errorf("bootstrap group %q is invalid (must match %s)", name, api.BootstrapGroupPattern)
}

// ValidateUsages validates that the passed in string are valid usage strings for bootstrap tokens.
func ValidateUsages(usages []string) error {
	validUsages := sets.NewString(api.KnownTokenUsages...)
	invalidUsages := sets.NewString()
	for _, usage := range usages {
		if !validUsages.Has(usage) {
			invalidUsages.Insert(usage)
		}
	}
	if len(invalidUsages) > 0 {
		return fmt.Errorf("invalid bootstrap token usage string: %s, valid usage options: %s", strings.Join(invalidUsages.List(), ","), strings.Join(api.KnownTokenUsages, ","))
	}
	return nil
}
