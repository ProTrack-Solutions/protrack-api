package validate

import (
	"errors"
	"unicode"
)

// ValidPassword valida se a senha do usuario tem mais de 8 caracteres
// E se contem letras e numeros
func ValidPassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}

	hasNumber := false
	hasLetter := false
	hasSpecialCharacter := false

	for _, c := range password {
		switch {
		case unicode.IsNumber(c):
			hasNumber = true
		case unicode.IsLetter(c):
			hasLetter = true
		case unicode.IsSymbol(c) || unicode.IsPunct(c):
			hasSpecialCharacter = true
		}
	}

	if !hasNumber || !hasLetter || !hasSpecialCharacter {
		return errors.New("password must contain at least one letter and one number and special Character")
	}

	return nil
}
