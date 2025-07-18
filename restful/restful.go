package restful

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"onlineChat/auth"
	bussinesslogic "onlineChat/bussinessLogic"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Restful struct {
	AuthLogic bussinesslogic.AuthBussinesslogic
}

func NewRestFul(authLogic bussinesslogic.AuthBussinesslogic) Restful {
	return Restful{
		AuthLogic: authLogic,
	}
}
func (rest Restful) Run() {
	//APIs
	http.HandleFunc("/signUp", rest.signUp)
	http.HandleFunc("/login", rest.login)
	http.HandleFunc("/verify", rest.smsVerification)
	http.HandleFunc("/room", rest.wsRoom)
	//listen
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
func (rest Restful) signUp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// اگر درخواست OPTIONS بود فقط پاسخ بده و برگرد
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	type User struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		PhoneNumber string `json:"phoneNumber"`
	}
	var user User
	//get user info from client
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	//insert user info into database
	err = rest.AuthLogic.SignUp(user.Username, user.Password, user.PhoneNumber)
	if err != nil {
		http.Error(w, "username or phone number already exist", 400)
	}

}

var verificationCodes = make(map[string]CodeInfo)

type CodeInfo struct {
	Code      string
	CreatedAt time.Time
}

var mu sync.Mutex // prevent Race Condition
func (rest Restful) login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// اگر درخواست OPTIONS بود فقط پاسخ بده و برگرد
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	type User struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	var user User
	//get login info from client
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	idStr, err := rest.AuthLogic.SendVerificationCode(user.Username, user.Password)
	if err != nil {
		http.Error(w, "username or password is wrong", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]string{"id": idStr, "username": user.Username})
	if err != nil {
		http.Error(w, "Failed to respond", http.StatusInternalServerError)
		return
	}
}

func (rest Restful) smsVerification(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	type Data struct {
		ID       string `json:"id"` // حالا string هست
		Code     string `json:"code"`
		Username string `json:"username"`
	}

	var data Data

	err := json.NewDecoder(r.Body).Decode(&data)
	fmt.Println("data: ", data)
	if err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	tokenStr, err := rest.AuthLogic.Verification(data.Code, data.ID, data.Username)
	if err != nil {
		fmt.Printf("reading error: %v ", err)
	}
	fmt.Println("token created in login: ", tokenStr)

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]string{"token": tokenStr})
	if err != nil {
		http.Error(w, "Failed to respond", http.StatusInternalServerError)
		return
	}

}

var broadcast = make(chan ChatMessage)
var clients = make(map[*websocket.Conn]bool)

type ChatMessage struct {
	Sender  string    `json:"sender"`
	Message string    `json:"message"`
	Date    time.Time `json:"date"`
}

func (rest Restful) wsRoom(w http.ResponseWriter, r *http.Request) {
	//upgrade http to ws
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("error in upgrade: %v", err)
		return
	}
	defer conn.Close()

	//get msg(token) first(one time)
	_, msg, err := conn.ReadMessage()
	if err != nil {
		log.Println("reading error", err)
		return
	}
	tokenStr := string(msg)
	// //put jwt info the claims and cheack validation
	// claims := &auth.Claims{}
	// tkn, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
	// 	return auth.Jwtkey, nil
	// })

	claims, err := auth.JWTvalidation(tokenStr)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"invalid token"}`))

		return
	}

	//if everything was ok make conn for user
	mu.Lock()
	clients[conn] = true
	mu.Unlock()

	go broadcastFunc()
	allMessages, err := rest.AuthLogic.SendMessages()
	marshalMessages, err := json.Marshal(allMessages)
	if err != nil {
		fmt.Printf("reading error: %v", err)
	}
	mu.Lock()
	for client := range clients {
		err = client.WriteMessage(websocket.TextMessage, marshalMessages)
		if err != nil {
			log.Println("write error:", err)
			client.Close()
			delete(clients, client)
		}
	}
	mu.Unlock()

	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			log.Printf("read error: %v", err)

			mu.Lock()
			delete(clients, conn)
			mu.Unlock()

			break
		}
		msg := string(msgBytes)
		err = rest.AuthLogic.SaveMessages(claims.Username, msg)
		if err != nil {
			panic(err)
		}

		chatMsg := ChatMessage{
			Sender:  claims.Username,
			Message: msg,
			Date:    time.Now(),
		}
		broadcast <- chatMsg
	}
}

func broadcastFunc() {
	for {
		chatMsg := <-broadcast
		jsonMsg, _ := json.Marshal(chatMsg)

		mu.Lock()
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, jsonMsg)
			if err != nil {
				log.Println("write error:", err)
				client.Close()
				delete(clients, client)
			}
		}
		mu.Unlock()
	}
}
