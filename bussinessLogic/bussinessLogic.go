package bussinesslogic

import (
	"errors"
	"fmt"
	"onlineChat/auth"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
)

type Imessage interface {
	SendMessage(reciever string) (string, error)
}
type ChatMessages struct {
	Username string    `bson:"username"`
	Message  string    `bson:"message"`
	Date     time.Time `bson:"time"`
}
type Idatabase interface {
	InsertUser(name, pass, phoneNumber string) error
	CheackUserById(name, pass string) (any, string, error)
	InsertMessagesIntoDatabase(username, msg string) error
	ReadAllMessagesFromDatabase() ([]ChatMessages, error)
}
type AuthBussinesslogic struct {
	Database Idatabase
	Message  Imessage
}

func NewAuthBussinessLogic(database Idatabase, mesesage Imessage) AuthBussinesslogic {
	return AuthBussinesslogic{
		Database: database,
		Message:  mesesage,
	}
}

type CodeInfo struct {
	Code      string
	CreatedAt time.Time
}

var verificationCodes = make(map[string]CodeInfo)

var mu sync.Mutex // prevent Race Condition
func (b AuthBussinesslogic) SendVerificationCode(username, password string) (string, error) {

	id, PhoneNumber, err := b.Database.CheackUserById(username, password)
	if err != nil {
		return "", err
	}
	code, err := b.Message.SendMessage(PhoneNumber)
	if err != nil {
		return "", err
	}
	idStr := fmt.Sprint(id)
	//lock the map while we add something to it so it prevent Race Condition
	mu.Lock()
	verificationCodes[idStr] = CodeInfo{
		Code:      code,
		CreatedAt: time.Now(),
	}
	mu.Unlock()
	return idStr, nil
}
func (b AuthBussinesslogic) SignUp(username, password, phoneNumber string) error {
	err := b.Database.InsertUser(username, password, phoneNumber)
	if err != nil {
		return err
	}
	return nil
}
func (b AuthBussinesslogic) Verification(code, id, username string) (string, error) {
	mu.Lock()
	//see if user is in map and if it is get the info for the specefic user (info -> code and time )
	info, ok := verificationCodes[id]
	mu.Unlock()
	if !ok {
		return "", errors.New("error")
	}

	if time.Since(info.CreatedAt) > 2*time.Minute {
		mu.Lock()
		delete(verificationCodes, id)
		mu.Unlock()

		return "", errors.New("error")
	}

	if code != info.Code {

		return "", errors.New("error")
	}
	//delete it so cant be used again
	mu.Lock()
	delete(verificationCodes, id)
	mu.Unlock()
	//if everything was ok generate JWT
	tokenStr := auth.GenerateJWT(id, username)
	return tokenStr, nil

}

var broadcast = make(chan ChatMessage)

type ChatMessage struct {
	Sender  string `json:"sender"`
	Message string `json:"message"`
}
type Claims struct {
	Username string `json:"username"`
	Id       any    `json:"id"`
	jwt.StandardClaims
}

//	func (b AuthBussinesslogic) TokenValidation(token string) (Claims, error) {
//		JWTinfo, err := auth.JWTvalidation(token)
//		if err != nil {
//			return Claims{}, err
//		}
//		return JWTinfo, nil
//	}
func (b AuthBussinesslogic) SaveMessages(username, msg string) error {
	err := b.Database.InsertMessagesIntoDatabase(username, msg)
	if err != nil {
		return err
	}
	return nil
}
func (b AuthBussinesslogic) SendMessages() ([]ChatMessages, error) {
	messages, err := b.Database.ReadAllMessagesFromDatabase()
	if err != nil {
		return []ChatMessages{}, err
	}
	return messages, nil
}
