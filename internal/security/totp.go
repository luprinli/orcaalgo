package security

import (
	"fmt"
	"strings"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

type TOTPSecret struct {
	Secret  string `json:"secret"`
	Issuer  string `json:"issuer"`
	Account string `json:"account"`
	Digits  int    `json:"digits"`
	Period  int64  `json:"period"`
}

func GenerateTOTPSecret(issuer, account string) *TOTPSecret {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: account,
	})
	if err != nil {
		key, _ = totp.Generate(totp.GenerateOpts{
			Issuer:      issuer,
			AccountName: account,
		})
	}
	return &TOTPSecret{
		Secret:  key.Secret(),
		Issuer:  issuer,
		Account: account,
		Digits:  6,
		Period:  30,
	}
}

func ValidateTOTP(secret string, code string) (bool, error) {
	if len(code) != 6 {
		return false, fmt.Errorf("TOTP code must be 6 digits")
	}
	secret = strings.ToUpper(secret)
	valid := totp.Validate(code, secret)
	return valid, nil
}

func GetTOTPURI(secret *TOTPSecret) string {
	key, err := otp.NewKeyFromURL(
		fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=%d&period=%d",
			secret.Issuer, secret.Account, secret.Secret, secret.Issuer, secret.Digits, secret.Period),
	)
	if err != nil {
		return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=%d&period=%d",
			secret.Issuer, secret.Account, secret.Secret, secret.Issuer, secret.Digits, secret.Period)
	}
	return key.URL()
}
