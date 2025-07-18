package main

import (
	"fmt"
	"net/http"
	"onlineChat/auth"
	bussinesslogic "onlineChat/bussinessLogic"
	"onlineChat/database"
	"onlineChat/restful"
)

func main() {
	var db database.Idatabase

	Mongo, err := database.NewMongoDB("mongodb://localhost:27017", "onlineChatDB")
	var LoginVerificationType auth.Imessage
	sms := auth.NewSMS()
	email := auth.NewEmail()
	//choosing database
	useMongo := true
	if useMongo {
		db = Mongo
	}
	//ChoosingLoginVerificationType
	useSMS := false
	if useSMS {
		LoginVerificationType = sms
	} else {
		LoginVerificationType = email
	}
	fmt.Println(LoginVerificationType)

	//myLogin := my.NewMyLogin()
	authLogic := bussinesslogic.NewAuthBussinessLogic(db, LoginVerificationType)
	r := restful.NewRestFul(authLogic)
	r.Run()

	//listen
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
