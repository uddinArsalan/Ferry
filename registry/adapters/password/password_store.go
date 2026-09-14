package password

type PasswordStore interface {
	HashAndEncodePassword(password string) (string, error)
	ComparePasswordAndHash(userPassword string, hash string) (bool, error)
}
