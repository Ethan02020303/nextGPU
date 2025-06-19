package process

import (
	log4plus "github.com/nextGPU/include/log4go"
)

const (
	//task
	UserSyncTask = int64(1)
)

type MatchHandle struct {
	selfHandle uint64
	wsIP       string
	wsPort     string
}

type UserMatch struct {
	session       *UserSession
	userLinker    []MatchHandle
	CloseCh       chan bool
	RecvMessageCh chan []byte
	SendMessageCh chan []byte
}

func (f *UserMatch) SetHandle(selfHandle uint64, ip, port string) {
	f.userLinker = append(f.userLinker, MatchHandle{selfHandle, ip, port})
}

func (f *UserMatch) clear() {
	f.userLinker = f.userLinker[:0]
}

func (f *UserMatch) SelfHandle() []uint64 {
	var handles []uint64
	for _, v := range f.userLinker {
		handles = append(handles, v.selfHandle)
	}
	return handles
}

func (f *UserMatch) IpPort() ([]string, []string) {
	var Ips []string
	var ports []string
	for _, v := range f.userLinker {
		Ips = append(Ips, v.wsIP)
		ports = append(ports, v.wsPort)
	}
	return Ips, ports
}

func (f *UserMatch) transfer() {
	for {
		select {
		case sendMsg := <-f.SendMessageCh:
			f.sendMessage(sendMsg)
		case <-f.CloseCh:
			f.closeUser()
			return
		}
	}
}

func (f *UserMatch) closeUser() {
	funName := "closeUser"
	log4plus.Info("%s closeCh ---->>>>", funName)
	f.clear()
}

func (f *UserMatch) sendMessage(message []byte) {
	for _, net := range f.session.Net {
		if net != nil {
			_ = net.WriteMessage(message)
		}
	}
}

func (f *UserMatch) UserPublish(message []byte) (error, TaskPublishResponse) {
	return userTaskPublish(f.session.Session, message)
}

func (f *UserMatch) UserStop(message []byte) (error, ResultMessage) {
	return userTaskStop(f.session.Session, message)
}

func (f *UserMatch) UserDelete(message []byte) (error, ResultMessage) {
	return userTaskDelete(f.session.Session, message)
}

func NewUserMatch(session *UserSession) *UserMatch {
	match := &UserMatch{
		session:       session,
		CloseCh:       make(chan bool),
		SendMessageCh: make(chan []byte, 1024),
		RecvMessageCh: make(chan []byte, 1024),
	}
	go match.transfer()
	return match
}
