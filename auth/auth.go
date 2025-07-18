package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/kavenegar/kavenegar-go"
	"gopkg.in/gomail.v2"
)

type Imessage interface {
	SendMessage(reciever string) (string, error)
}
type SMS struct {
}

func NewSMS() SMS {
	return SMS{}
}

func (s SMS) SendMessage(receiver string) (string, error) {
	code, err := SendSMSVerification(receiver)
	fmt.Println("using sms")
	if err != nil {
		return "", err
	}
	return code, nil

}

func SendSMSVerification(to string) (string, error) {
	code, err := GenerateSecureCode()
	if err != nil {
		panic(err)
	}
	api := kavenegar.New("625434516D734A442B506C476A313266676D715A57772B4A785431516F66624F35496A716A6E504C5936733D")
	sender := "2000660110" // شماره ارسال‌کننده‌ی مجاز

	res, err := api.Message.Send(sender, []string{to}, fmt.Sprintf("کد تایید شما: %s", code), nil)
	if err != nil {
		fmt.Println("خطا:", err)
		return "", err
	}

	fmt.Println("ارسال شد به:", res[0].MessageID)
	return code, nil

}

type Email struct{}

func NewEmail() Email {
	return Email{}
}
func (e Email) SendMessage(reciever string) (string, error) {
	fmt.Println("using email")
	code, err := SendEmailVerificationCode(reciever)
	if err != nil {
		return "", err
	}
	return code, nil
}
func SendEmailVerificationCode(reciever string) (string, error) {
	code, err := GenerateSecureCode()
	if err != nil {
		panic(err)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", "pouriyahmn@gmail.com")
	m.SetHeader("To", reciever)
	m.SetHeader("Subject", "Your Verification Code")
	m.SetBody("text/plain", "Your verification code is: "+code)

	d := gomail.NewDialer("smtp.gmail.com", 587, "pouriyahmn@gmail.com", "yezs zujy czwx xiew")

	err = d.DialAndSend(m)
	if err != nil {
		return "", err
	}

	return code, nil
}
func GenerateSecureCode() (string, error) {
	max := big.NewInt(1000000) // ۶ رقم
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

type Claims struct {
	Username string `json:"username"`
	Id       any    `json:"id"`
	jwt.StandardClaims
}

var Jwtkey = []byte("secret-key")

func GenerateJWT(id any, username string) string {

	var jwtkey = []byte("secret-key")

	expireTime := time.Now().Add(time.Minute * 100)
	claims := &Claims{
		Username: username,

		Id: id,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expireTime.Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtkey)
	if err != nil {
		panic(err)
	}
	return tokenString
}

func JWTvalidation(tokenStr string) (Claims, error) {
	claims := &Claims{}
	tkn, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return Jwtkey, nil
	})
	if err != nil || !tkn.Valid {

		return Claims{}, err
	}

	return *claims, nil
}
