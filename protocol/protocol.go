package protocol

import (
	"net/http"
	"onlineChat/database"
)

type WsProtocol struct {
	DB database.DatabaseAdapter
}
type HttpProtocol struct {
	DB database.DatabaseAdapter
}
type Iprotocol interface {
	Login(w http.ResponseWriter, r *http.Request)
	SignUp(w http.ResponseWriter, r *http.Request)
}
