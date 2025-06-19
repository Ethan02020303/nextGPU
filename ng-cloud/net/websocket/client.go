package ws

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	log4plus "github.com/nextGPU/include/log4go"
	"net"
	"sync"
)

type WSClient struct {
	wsUrl        string
	param        string
	event        OnEvent
	lock         sync.Mutex
	linkers      map[uint64]*Linker
	handleLock   sync.Mutex
	startHandle  uint64
	curHandle    uint64
	disconnectCh chan uint64
	mode         NetMode
}

func (w *WSClient) addLinker(linker *Linker) {
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

func (w *WSClient) findLinker(handler uint64) *Linker {
	w.lock.Lock()
	defer w.lock.Unlock()
	linker, Ok := w.linkers[handler]
	if Ok {
		return linker
	}
	return nil
}

func (w *WSClient) createHandler() uint64 {
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

func (w *WSClient) StartClient() *Linker {
	funName := "init"
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(w.wsUrl, nil)
	if err != nil {
		errString := fmt.Sprintf("%s dialer.Dial Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return nil
	}
	remoteIp := ""
	remotePort := ""
	if remoteIp, remotePort, err = net.SplitHostPort(conn.RemoteAddr().String()); err != nil {
		errString := fmt.Sprintf("%s SplitHostPort address Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return nil
	}
	log4plus.Info("%s New Connection remoteIp=[%s] remotePort=[%s]---->>>>", funName, remoteIp, remotePort)
	handle := w.createHandler()
	linker := NewLinker(w.mode, w.param, handle, remoteIp, remotePort, w.event, conn)
	w.addLinker(linker)
	return linker
}

func (w *WSClient) SendMessage(handle uint64, message []byte) error {
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

func NewWSClient(wsUrl string, startHandle uint64, event OnEvent) *WSClient {
	gWebSocket := &WSClient{
		wsUrl:        wsUrl,
		mode:         NodeMode,
		curHandle:    startHandle,
		linkers:      make(map[uint64]*Linker),
		disconnectCh: make(chan uint64, 1024),
		event:        event,
	}
	return gWebSocket
}
