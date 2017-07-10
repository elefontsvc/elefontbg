package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

//Message represents the struct that's sent between the electron client and the service binary
type Message struct {
	Version int    `json:"version"`
	Type    int    `json:"type"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

const (
	//AddFont tells the service to add a specific font to the user space
	AddFont = iota
	//DelFont tells the service to remove a font from the user space
	DelFont
	//GetFont tells the service to list all available fonts (installed and uninstalled)
	GetFont
	//Hearbeat is a heartbeat message
	Heartbeat
	//Unknown is to tell the endpoints there's no way of knowing what is going on
	Unknown
)

// https://jacobmartins.com/2016/03/07/practical-golang-using-websockets/
// https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API/Writing_WebSocket_client_applications
// https://discuss.atom.io/t/how-to-pass-more-than-one-function-in-a-js-file-to-another-file/33134/4
// http://www.gorillatoolkit.org/pkg/websocket
const (
	//StatusOK means the command/request completed successfully and the payload can be found in the message-field
	StatusOK = iota
	//StatusWait means the service is still performing the request
	StatusWait
	//StatusFailed means the service failed to perform the request and further info can be found in the message
	StatusFailed
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("%v", err)
		return
	}
	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		}

		mess := Message{}
		err = json.Unmarshal(msg, &mess)
		if err != nil {
			log.Printf("could not unmarshal json (%v)", err)
			continue
		}
		ans := answer(&mess)
		// ans := Message{
		// 	Type:    mess.Type,
		// 	Status:  StatusOK,
		// 	Message: "this is a response",
		// 	Version: 1,
		// }
		ans.Version = 1 //Currently, this is the only supported protocol

		log.Printf("rcv: '%+v'", mess)

		b, err := json.Marshal(ans)
		if err != nil {
			log.Printf("could not marshal response: %v", err)
			continue
		}

		err = conn.WriteMessage(msgType, b)
		if err != nil {
			log.Printf("%v", err)
			return
		}
	}
}

func answer(m *Message) *Message {
	ans := &Message{}
	if m.Type == GetFont {
		ans.Type = GetFont
		ans.Message = "Should contain a list of all available fonts, with their respective ID:s"
		return ans
	}

	if m.Type == AddFont {
		ans.Type = AddFont
		ans.Message = "Should contain just a message or something"
		ans.Status = StatusOK
		return ans
	}
	ans.Type = Unknown
	ans.Message = "Unrecognized type"
	ans.Status = StatusFailed
	return ans
}
