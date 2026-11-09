package hash

import "github.com/alexedwards/argon2id"

func Hash(password string) (string, error) {
	return argon2id.CreateHash(password, argon2id.DefaultParams)
}

func Verify(password, hashed string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hashed)
}
