package helpers

import (
	"github.com/mcnijman/go-emailaddress"
	"github.com/pkg/errors"
)

const MaxEmailLength = 254

func ValidateEmail(email string) (string, error) {
	parsedEmail, err := emailaddress.Parse(email)
	if err != nil {
		return "", errors.Wrap(err, "couldn't parse email")
	}

	email = parsedEmail.String()

	if len(email) > MaxEmailLength {
		return "", errors.New("email too long")
	}

	return email, nil
}
