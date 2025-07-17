package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/kavenegar/kavenegar-go"
)

func SendSMSVerification(to string) (string, error) {
	code, err := generateSecureCode()
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
func generateSecureCode() (string, error) {
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
