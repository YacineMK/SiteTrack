package hash

import "github.com/alexedwards/argon2id"

var params = &argon2id.Params{
	Memory:      32 * 1024,
	Iterations:  3,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}

func Hash(password string) (string, error) {
	return argon2id.CreateHash(password, params)
}

func Verify(password, hashed string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, hashed)
}
