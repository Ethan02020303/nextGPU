package ws

import (
	"errors"
	"fmt"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"strings"
	"sync"
)

//_____________________________________________________

type WSServer struct {
	routePath   string
	listen      string
	event       OnEvent
	lock        sync.Mutex
	linkers     map[uint64]*Linker
	handleLock  sync.Mutex
	curHandle   uint64
	startHandle uint64
	//timeOutSecond int
	disconnectCh chan uint64
	mode         NetMode
}

func (w *WSServer) addLinker(linker *Linker) {
	funName := "addLinker"
	if linker == nil {
		errString := fmt.Sprintf("%s linker is nil", funName)
		log4plus.Error(errString)
		return
	}
	w.lock.Lock()
	defer w.lock.Unlock()
	w.linkers[linker.Handle()] = linker
}

func (w *WSServer) deleteLinker(handle uint64) {
	w.lock.Lock()
	defer w.lock.Unlock()
	linker, Ok := w.linkers[handle]
	if Ok {
		_ = linker.Close()
		delete(w.linkers, handle)
	}
}

func (w *WSServer) findLinker(handler uint64) *Linker {
	w.lock.Lock()
	defer w.lock.Unlock()
	linker, Ok := w.linkers[handler]
	if Ok {
		return linker
	}
	return nil
}

func (w *WSServer) createHandler() uint64 {
	w.handleLock.Lock()
	defer w.handleLock.Unlock()

	for {
		w.curHandle++
		if w.curHandle < w.startHandle {
			w.curHandle = w.startHandle
		}
		linker := w.findLinker(w.curHandle)
		if linker == nil {
			return w.curHandle
		}
	}
}

func (w *WSServer) upgrader() *websocket.Upgrader {
	var tmpUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	return &tmpUpgrader
}

func (w *WSServer) handle(writer http.ResponseWriter, r *http.Request) {
	funName := "handle"
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		errString := fmt.Sprintf("%s Invalid path err=[%d]", funName, http.StatusBadRequest)
		log4plus.Error(errString)
		http.Error(writer, "Invalid path", http.StatusBadRequest)
		return
	}
	param := ""
	if w.mode == NodeMode {
		param = parts[2]
		if exist := w.event.Exist(param); !exist {
			http.Error(writer, "Invalid path", http.StatusBadRequest)
			return
		}
	} else if w.mode == UserMode {
		param = parts[2]
		if exist := w.event.Exist(param); !exist {
			errString := fmt.Sprintf("%s user not found param=[%s]", funName, param)
			log4plus.Error(errString)
			http.Error(writer, "Invalid path", http.StatusBadRequest)
			return
		}
	}
	conn, err := w.upgrader().Upgrade(writer, r, nil)
	if err != nil {
		errString := fmt.Sprintf("%s upgrader.Upgrade Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	remoteIp := ""
	remotePort := ""
	if remoteIp, remotePort, err = net.SplitHostPort(conn.RemoteAddr().String()); err != nil {
		errString := fmt.Sprintf("%s SplitHostPort address Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return
	}
	handle := w.createHandler()
	linker := NewLinker(w.mode, param, handle, remoteIp, remotePort, w.event, conn)
	w.addLinker(linker)
	w.event.OnConnect(linker, param, remoteIp, remotePort)
}

func (w *WSServer) StartServer() {
	funName := "StartServer"
	route := fmt.Sprintf("%s/{%s}", w.routePath, "nodeId")
	http.HandleFunc(route, w.handle)
	if err := http.ListenAndServe(w.listen, nil); err != nil {
		errString := fmt.Sprintf("%s http.ListenAndServe Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
	}
	log4plus.Info("%s success---->>>>", funName)
}

func (w *WSServer) SendMessage(handle uint64, message []byte) error {
	funName := "SendMessage"
	linker := w.findLinker(handle)
	if linker != nil {
		if err := linker.WriteMessage(message); err != nil {
			errString := fmt.Sprintf("%s WriteMessage Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
		}
		return nil
	}
	return errors.New("not find linker")
}

func NewWSServer(routePath, listen string, mode NetMode, startHandle uint64, event OnEvent) *WSServer {
	gWebSocket := &WSServer{
		curHandle:    startHandle,
		linkers:      make(map[uint64]*Linker),
		disconnectCh: make(chan uint64, 1024),
		mode:         mode,
		routePath:    routePath,
		listen:       listen,
		event:        event,
	}
	return gWebSocket
}
