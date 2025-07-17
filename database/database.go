package database

type Idatabase interface {
	InsertUser(name, pass, phoneNumber string) error
	CheackUserById(name, pass string) (any, string, error)
}
