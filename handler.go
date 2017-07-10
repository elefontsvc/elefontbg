package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/gorilla/websocket"
)

//Message represents the struct that's sent between the electron client and the service binary
type Message struct {
	Version int    `json:"version"`
	Type    int    `json:"type"`
	Message string `json:"message"`
	Status  int    `json:"status"`
	Fonts   []Font `json:"fonts,omitempty"`
}

const (
	//AddFont tells the service to add a specific font to the user space
	AddFont = iota
	//DelFont tells the service to remove a font from the user space
	DelFont
	//GetFont tells the service to list all available fonts (installed and uninstalled)
	GetFont
	//Heartbeat is a heartbeat message
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
		for _, f := range installedFonts {
			ans.Fonts = append(ans.Fonts, f)
		}
		ans.Type = GetFont
		ans.Message = ""
		ans.Status = StatusOK
		return ans
	}

	if m.Type == AddFont {
		//Copy the file to the elefontdir
		log.Printf("adding font")
		ans.Type = AddFont
		if !(len(m.Fonts) > 0) {
			ans.Status = StatusFailed
			ans.Message = "No fonts were selected"
			return ans
		}

		f, err := os.Open(m.Fonts[0].Path) //we only support one font at a time right now
		if err != nil {
			log.Printf("%v", err)
			ans.Status = StatusFailed
			ans.Message = fmt.Sprintf("%v", err)
			return ans
		}
		defer f.Close()
		dstpath := fmt.Sprintf("%s/%s", elefontDir, filepath.Base(f.Name()))
		dst, err := os.Create(dstpath)
		if err != nil {
			log.Printf("%v", err)
			ans.Status = StatusFailed
			ans.Message = fmt.Sprintf("%v", err)
			return ans
		}
		defer dst.Close()

		_, err = io.Copy(dst, f)
		if err != nil {
			log.Printf("%v", err)
			ans.Status = StatusFailed
			ans.Message = fmt.Sprintf("%v", err)
			return ans
		}

		err = dst.Sync()
		if err != nil {
			log.Printf("%v", err)
			ans.Status = StatusFailed
			ans.Message = fmt.Sprintf("%v", err)
			return ans
		}
		err = installFont(dstpath)
		if err != nil {
			ans.Status = StatusFailed
			ans.Message = err.Error()
			return ans
		}
		ans.Message = fmt.Sprintf("Font %s installed", f.Name())
		ans.Status = StatusOK
		log.Printf("added OK!")
		loadInstalledFonts()
		return ans
	}

	if m.Type == DelFont {
		ans.Type = DelFont
		log.Printf("uninstalling font")
		if !(len(m.Fonts) > 0) {
			ans.Status = StatusFailed
			ans.Message = "No font was selected"
			return ans
		}
		fid, ok := installedFonts[m.Fonts[0].ID]
		log.Printf("%v", fid)
		for _, ff := range installedFonts {
			log.Printf("'%s' vs '%s'", ff.ID, m.Fonts[0].ID)
		}

		if !ok {
			ans.Status = StatusFailed
			ans.Message = fmt.Sprintf("File %s could not be found", m.Fonts[0].Path)
			return ans
		}

		err := uninstallFont(fid.Path)
		log.Printf("uninstall err: %v", err)
		if err != nil {
			ans.Status = StatusFailed
			ans.Message = err.Error()
			return ans
		}
		time.Sleep(time.Millisecond * 500)
		err = os.Remove(fid.Path)
		if err != nil {
			ans.Status = StatusFailed
			ans.Message = fmt.Sprintf("%v", err)
			return ans
		}

		ans.Status = StatusOK
		ans.Message = fmt.Sprintf("%s was uninstalled", fid.Name)
		loadInstalledFonts()
		return ans
	}

	ans.Type = Unknown
	ans.Message = "Unrecognized type"
	ans.Status = StatusFailed
	return ans
}

func installFont(font string) error {
	// https://msdn.microsoft.com/en-us/library/windows/desktop/dd183326(v=vs.85).aspx
	mod := syscall.NewLazyDLL("Gdi32.dll")
	proc := mod.NewProc("AddFontResourceW")
	_, _, err := proc.Call(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(font))))
	// log.Printf("%v, %v, %v", ret, ret2, err)
	return completedSuccessfully(err)
}

func uninstallFont(font string) error {
	// https://msdn.microsoft.com/en-us/library/windows/desktop/dd162922(v=vs.85).aspx
	mod := syscall.NewLazyDLL("Gdi32.dll")
	proc := mod.NewProc("RemoveFontResourceW")
	_, _, err := proc.Call(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(font))))
	// log.Printf("%v, %v, %v", ret, ret2, err)
	return completedSuccessfully(err)
}

func completedSuccessfully(err error) error {
	if strings.Compare("The operation completed successfully.", err.Error()) == 0 {
		return nil
	}
	return err
}
