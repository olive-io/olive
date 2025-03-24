package jwtutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateToken(t *testing.T) {
	token, err := GenerateToken(1, "aa")
	if err != nil {
		t.Fatal(err)
	}

	token1, err := GenerateToken(2, "bb")
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEqual(t, token, token1)
}

func TestParseToken(t *testing.T) {
	id := uint64(1)
	secret := "aa"
	token, err := GenerateToken(id, secret)
	if err != nil {
		t.Fatal(err)
	}

	claims, err := ParseToken(token.Token)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, id, claims.Identified)
	assert.Equal(t, secret, claims.Secret)
}
