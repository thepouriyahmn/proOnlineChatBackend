package database

import bussinesslogic "onlineChat/bussinessLogic"

type Idatabase interface {
	InsertUser(name, pass, phoneNumber string) error
	CheackUserById(name, pass string) (any, string, error)
	InsertMessagesIntoDatabase(username, msg string) error
	ReadAllMessagesFromDatabase() ([]bussinesslogic.ChatMessages, error)
}
