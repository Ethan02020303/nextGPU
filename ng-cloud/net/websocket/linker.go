package ws

import (
	"errors"
	"fmt"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/gorilla/websocket"
	"time"
)

type OnEvent interface {
	OnConnect(linker *Linker, param string, Ip string, port string)
	OnRead(linker *Linker, param string, message []byte) error
	OnDisconnect(linker *Linker, param string, Ip string, port string)
	Exist(param string) bool
}

const (
	HeartTimeout = int64(15) //Measurement Tasks
)

type NetMode uint8

const (
	NodeMode NetMode = iota + 1
	UserMode
)

type Linker struct {
	heartbeatCh chan struct{}
	exitCh      chan struct{}
	handle      uint64
	remoteIp    string
	remotePort  string
	heartbeat   time.Time
	conn        *websocket.Conn
	event       OnEvent
	mode        NetMode
	param       string
}

func (l *Linker) Handle() uint64 {
	return l.handle
}

func (l *Linker) Ip() string {
	return l.remoteIp
}

func (l *Linker) Port() string {
	return l.remotePort
}

func (l *Linker) recv() {
	funName := "recv"
	for {
		messageType, message, err := l.conn.ReadMessage()
		if err != nil {
			errString := fmt.Sprintf("%s w.conn.ReadMessage Failed mode=[%d] handle=[%d] err=[%s]", funName, l.mode, l.Handle(), err.Error())
			log4plus.Error(errString)
			l.event.OnDisconnect(l, l.param, l.Ip(), l.Port())
			close(l.exitCh)
			return
		}
		l.heartbeatCh <- struct{}{}
		if messageType == websocket.CloseMessage {
			errString := fmt.Sprintf("%s messageType is websocket.CloseMessage", funName)
			log4plus.Error(errString)
			l.event.OnDisconnect(l, l.param, l.Ip(), l.Port())
			close(l.exitCh)
			return
		} else if messageType == websocket.PingMessage {
			l.heartbeat = time.Now()
			continue
		}
		_ = l.event.OnRead(l, l.param, message)
	}
}

func (l *Linker) ping(data string) error {
	l.heartbeatCh <- struct{}{}
	return l.conn.WriteControl(websocket.PongMessage, []byte(data), time.Now().Add(time.Second))
}

func (l *Linker) WriteJSON(v interface{}) error {
	funName := "WriteJSON"
	if l.conn != nil {
		if err := l.conn.WriteJSON(v); err != nil {
			errString := fmt.Sprintf("%s conn.WriteJSON Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			return err
		}
		return nil
	}
	return errors.New(fmt.Sprintf("%s w.conn is nil", funName))
}

func (l *Linker) WriteMessage(data []byte) error {
	funName := "WriteMessage"
	if l.conn != nil {
		messageType := websocket.TextMessage
		if err := l.conn.WriteMessage(messageType, data); err != nil {
			errString := fmt.Sprintf("%s conn.WriteMessage Failed err=[%s]", funName, err.Error())
			log4plus.Error(errString)
			return err
		}
		return nil
	}
	return errors.New(fmt.Sprintf("%s w.conn is nil", funName))
}

func (l *Linker) Close() error {
	funName := "WriteMessage"
	if l.conn != nil {
		_ = l.conn.Close()
	}
	return errors.New(fmt.Sprintf("%s w.conn is nil", funName))
}

func (l *Linker) checkTimeOut() {
	funName := "checkTimeOut"
	for {
		select {
		case <-l.heartbeatCh:
			l.heartbeat = time.Now()
		//case <-time.After(time.Duration(HeartTimeout) * time.Second):
		//	log4plus.Info("%s heart timeout handle=[%d]", funName, l.handle)
		//	_ = l.conn.Close()
		//	return
		case <-l.exitCh:
			log4plus.Info("%s exitCh", funName)
			return
		}
	}
}

func NewLinker(mode NetMode, param string, handle uint64, remoteIp string, remotePort string, event OnEvent, conn *websocket.Conn) *Linker {
	linker := &Linker{
		handle:      handle,
		remoteIp:    remoteIp,
		remotePort:  remotePort,
		conn:        conn,
		event:       event,
		mode:        mode,
		param:       param,
		heartbeat:   time.Now(),
		heartbeatCh: make(chan struct{}),
		exitCh:      make(chan struct{}),
	}
	conn.SetPingHandler(linker.ping)
	go linker.recv()
	go linker.checkTimeOut()
	return linker
}
