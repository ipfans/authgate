package models

import (
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/sqids/sqids-go"
)

var _ webauthn.User = &User{}

var idGenerator *sqids.Sqids

type User struct {
	ID          uint64 `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

func init() {
	var err error
	idGenerator, err = sqids.New()
	if err != nil {
		panic(err)
	}
}

func (u *User) WebAuthnID() []byte {
	id, err := idGenerator.Encode([]uint64{u.ID})
	if err != nil {
		panic(err)
	}
	return []byte(id)
}

func (u *User) WebAuthnCredentials() []webauthn.Credential {
	return []webauthn.Credential{}
}

func (u *User) WebAuthnDisplayName() string {
	return u.DisplayName
}

func (u *User) WebAuthnName() string {
	return u.Username
}
