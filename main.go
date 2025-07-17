package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"onlineChat/auth"
	"onlineChat/database"
	"sync"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/websocket"
)

var db database.Idatabase

func main() {
	Mongo, err := database.NewMongoDB("mongodb://localhost:27017", "onlineChatDB")

	useMongo := true
	if useMongo {
		db = Mongo
	}

	http.HandleFunc("/signUp", signUp)
	http.HandleFunc("/login", login)
	http.HandleFunc("/verify", smsVerification)
	http.HandleFunc("/room", wsRoom)
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
func signUp(w http.ResponseWriter, r *http.Request) {
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
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	err = db.InsertUser(user.Username, user.Password, user.PhoneNumber)
	if err != nil {
		http.Error(w, "username or phone number already exist", 400)
	}

}

// var verificationCodes = make(map[any]CodeInfo)
var verificationCodes = make(map[string]CodeInfo) // ID از نوع string

type CodeInfo struct {
	Code      string
	CreatedAt time.Time
}

var mu sync.Mutex // برای جلوگیری از race condition
func login(w http.ResponseWriter, r *http.Request) {
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
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	id, phoneNumber, err := db.CheackUserById(user.Username, user.Password)
	if err != nil {
		http.Error(w, "username or password is wrong", http.StatusUnauthorized)
		return
	}

	code, err := auth.SendSMSVerification(phoneNumber)
	if err != nil {
		http.Error(w, "Failed to send verification code", http.StatusInternalServerError)
		return
	}

	idStr := fmt.Sprint(id) // تبدیل ID به string

	mu.Lock()
	verificationCodes[idStr] = CodeInfo{
		Code:      code,
		CreatedAt: time.Now(),
	}
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]string{"id": idStr, "username": user.Username})
	if err != nil {
		http.Error(w, "Failed to respond", http.StatusInternalServerError)
		return
	}
}

func smsVerification(w http.ResponseWriter, r *http.Request) {
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

	mu.Lock()
	info, ok := verificationCodes[data.ID]
	mu.Unlock()

	if !ok {
		http.Error(w, "no code found", http.StatusNotFound)
		return
	}

	if time.Since(info.CreatedAt) > 2*time.Minute {
		mu.Lock()
		delete(verificationCodes, data.ID)
		mu.Unlock()
		http.Error(w, "code expired", http.StatusUnauthorized)
		return
	}

	if data.Code != info.Code {
		http.Error(w, "invalid code", http.StatusUnauthorized)
		return
	}

	mu.Lock()
	delete(verificationCodes, data.ID)
	mu.Unlock()
	tokenStr := auth.GenerateJWT(data.ID, data.Username)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]string{"token": tokenStr})
	if err != nil {
		http.Error(w, "Failed to respond", http.StatusInternalServerError)
		return
	}

	//	w.WriteHeader(http.StatusOK)
	//	w.Write([]byte("verified"))
}

var broadcast = make(chan ChatMessage)
var clients = make(map[*websocket.Conn]bool)

type ChatMessage struct {
	Sender  string `json:"sender"`
	Message string `json:"message"`
}

func wsRoom(w http.ResponseWriter, r *http.Request) {

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	fmt.Println("api running")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("error in upgrade: %v", err)
		return
	}
	defer conn.Close()

	// دریافت توکن و اعتبارسنجی
	_, msg, err := conn.ReadMessage()
	if err != nil {
		log.Println("reading error", err)
		return
	}
	tokenStr := string(msg)

	claims := &auth.Claims{}
	tkn, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return auth.Jwtkey, nil
	})
	if err != nil || !tkn.Valid {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"invalid token"}`))

		return
	}

	// اضافه کردن به clients با قفل
	mu.Lock()
	clients[conn] = true
	mu.Unlock()

	go broadcastFunc() // فقط یک بار اجرا بشه، بهتره بیرون از wsRoom اجرا شه

	for {
		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			log.Printf("read error: %v", err)

			// حذف از clients با قفل
			mu.Lock()
			delete(clients, conn)
			mu.Unlock()

			break
		}
		msg := string(msgBytes)
		chatMsg := ChatMessage{
			Sender:  claims.Username,
			Message: msg,
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
