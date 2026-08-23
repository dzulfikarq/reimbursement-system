package password

import "golang.org/x/crypto/bcrypt"

func Hash(plain string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(bytes), err
}

func Compare(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}