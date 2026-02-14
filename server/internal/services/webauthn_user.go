package services

import (
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/naiba/bonds/internal/models"
)

// webAuthnUser adapts a models.User + credentials to the webauthn.User interface.
type webAuthnUser struct {
	user        *models.User
	credentials []models.WebAuthnCredential
}

func newWebAuthnUser(user *models.User, creds []models.WebAuthnCredential) *webAuthnUser {
	return &webAuthnUser{user: user, credentials: creds}
}

func (u *webAuthnUser) WebAuthnID() []byte {
	return []byte(u.user.ID)
}

func (u *webAuthnUser) WebAuthnName() string {
	return u.user.Email
}

func (u *webAuthnUser) WebAuthnDisplayName() string {
	name := ""
	if u.user.FirstName != nil {
		name = *u.user.FirstName
	}
	if u.user.LastName != nil {
		if name != "" {
			name += " "
		}
		name += *u.user.LastName
	}
	if name == "" {
		name = u.user.Email
	}
	return name
}

func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	creds := make([]webauthn.Credential, len(u.credentials))
	for i, c := range u.credentials {
		creds[i] = webauthn.Credential{
			ID:              c.CredentialID,
			PublicKey:       c.PublicKey,
			AttestationType: c.AttestationType,
			Authenticator: webauthn.Authenticator{
				AAGUID:    c.AAGUID,
				SignCount: c.SignCount,
			},
		}
	}
	return creds
}

func (u *webAuthnUser) WebAuthnIcon() string {
	return ""
}

// CredentialExcludeList returns credentials to exclude during registration.
func (u *webAuthnUser) CredentialExcludeList() []protocol.CredentialDescriptor {
	descriptors := make([]protocol.CredentialDescriptor, len(u.credentials))
	for i, c := range u.credentials {
		descriptors[i] = protocol.CredentialDescriptor{
			Type:         protocol.PublicKeyCredentialType,
			CredentialID: c.CredentialID,
		}
	}
	return descriptors
}
