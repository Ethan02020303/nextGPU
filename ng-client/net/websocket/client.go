package ws

import (
	"errors"
	"fmt"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/gorilla/websocket"
	"net"
	"sync"
)

type WSClient struct {
	//event         OnEvent
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

func (w *WSClient) deleteLinker(handle uint64) {
	w.lock.Lock()
	defer w.lock.Unlock()
	linker, Ok := w.linkers[handle]
	if Ok {
		_ = linker.Close()
		delete(w.linkers, handle)
	}
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

func (w *WSClient) StartClient(wsUrl, param string, event OnEvent) *Linker {
	funName := "init"
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(wsUrl, nil)
	if err != nil {
		errString := fmt.Sprintf("%s dialer.Dial Failed err=[%s] wsUrl=[%s]", funName, err.Error(), wsUrl)
		log4plus.Error(errString)
		return nil
	}
	//conn.SetReadLimit()
	//conn.SetReadDeadline()
	remoteIp := ""
	remotePort := ""
	if remoteIp, remotePort, err = net.SplitHostPort(conn.RemoteAddr().String()); err != nil {
		errString := fmt.Sprintf("%s SplitHostPort address Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return nil
	}
	log4plus.Info(fmt.Sprintf("%s New Connection remoteIp=[%s] remotePort=[%s]---->>>>", funName, remoteIp, remotePort))
	handle := w.createHandler()
	linker := NewLinker(w.mode, param, handle, remoteIp, remotePort, event, conn)
	linker.State = Connecting
	w.addLinker(linker)
	event.OnConnect(linker, param, remoteIp, remotePort)
	return linker
}

func (w *WSClient) SendMessage(handle uint64, message []byte) error {
	funName := "SendMessage"
	linker := w.findLinker(handle)
	if linker != nil && linker.conn != nil {
		if err := linker.WriteMessage(message); err != nil {
			errString := fmt.Sprintf("%s WriteMessage Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
		}
		return nil
	}
	return errors.New("not find linker")
}

func (w *WSClient) Close(handle uint64) {
	linker := w.findLinker(handle)
	if linker != nil {
		w.deleteLinker(handle)
	}
}

func NewWSClient(startHandle uint64) *WSClient {
	gWebSocket := &WSClient{
		mode:         NodeMode,
		startHandle:  startHandle,
		curHandle:    startHandle,
		linkers:      make(map[uint64]*Linker),
		disconnectCh: make(chan uint64, 1024),
	}
	return gWebSocket
}
