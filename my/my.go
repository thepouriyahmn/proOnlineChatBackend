package my

import (
	"time"
)

type User struct {
	UserName             string
	Password             string
	MobileNumber         string
	VerificationCode     string
	VerificationCodeDate time.Time
}

type UserRepository interface {
	GetUserByUserNameAndPassword(userName, password string) (User, error)
	SaveUser(user User) error
}

type NumberGenerator interface {
	Generate(len int) string
}

type Message interface {
	Send(receiver, message string) error
}

type MyLogic struct {
	repo    UserRepository
	number  NumberGenerator
	message Message
}

func NewMyLogin(repo UserRepository, number NumberGenerator, message Message) MyLogic {
	return MyLogic{
		repo:    repo,
		number:  number,
		message: message,
	}
}

func (m MyLogic) SendVerificationCode(userName, password string) error {
	user, err := m.repo.GetUserByUserNameAndPassword(userName, password)
	if err != nil {
		return err
	}
	verificationCode := m.number.Generate(5)
	err = m.message.Send(user.MobileNumber, verificationCode)
	if err != nil {
		return err
	}
	user.VerificationCode = verificationCode
	user.VerificationCodeDate = time.Now()
	err = m.repo.SaveUser(user)
	if err != nil {
		return err
	}
	return nil
}
