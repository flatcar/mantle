package brightbox

import "time"

// User represents a Brightbox User
// https://api.gb1.brightbox.com/1.0/#user
type User struct {
	ResourceRef
	ID             string
	Name           string
	EmailAddress   string            `json:"email_address"`
	EmailVerified  bool              `json:"email_verified"`
	SSHKey         string            `json:"ssh_key"`
	MessagingPref  bool              `json:"messaging_pref"`
	CreatedAt      *time.Time        `json:"created_at"`
	TwoFactorAuth  TwoFactorAuthType `json:"2fa"`
	DefaultAccount *Account          `json:"default_account"`
	Accounts       []Account
}

// TwoFactorAuthType is nested in User
type TwoFactorAuthType struct {
	Enabled bool
}

// UserOptions is used to update objects
type UserOptions struct {
	ID                   string  `json:"-"`
	Name                 *string `json:"name,omitempty"`
	EmailAddress         *string `json:"email_address,omitempty"`
	SSHKey               *string `json:"ssh_key,omitempty"`
	Password             *string `json:"password,omitempty"`
	PasswordConfirmation *string `json:"password_confirmation,omitempty"`
}
