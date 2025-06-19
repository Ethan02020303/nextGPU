package process

import (
	"fmt"
	"github.com/nextGPU/ng-cloud/db"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/jinzhu/copier"
	"sync"
)

type (
	WorkSpaceRequest struct {
		Area     string `json:"area"`
		NetBands int64  `json:"netBands"`
		Storage  int64  `json:"storage"`
		GPU      string `json:"gpu"`
		Measure  string `json:"measure"`
	}
	WorkSpace struct {
		Base        db.WorkSpaceBase `json:"base"`
		Request     WorkSpaceRequest `json:"request"`
		SystemUUIDs []string         `json:"systemUUIDs"`
	}
)

type WorkSpaces struct {
	lock       sync.Mutex
	workSpaces []*WorkSpace
}

var (
	gWorkSpaces *WorkSpaces
)

func (w *WorkSpaces) AllWorkSpaces() []WorkSpace {
	funName := "AllWorkSpaces"
	var workSpaces []WorkSpace
	if err, curWorkSpaces := db.SingletonNodeBaseDB().WorkSpaces(); err != nil {
		errString := fmt.Sprintf("%s WorkSpaces Failed err=[%s]", funName, err.Error())
		log4plus.Error(errString)
		return workSpaces
	} else {
		for _, v := range curWorkSpaces {
			cur := WorkSpace{}
			_ = copier.Copy(&cur.Base, v)
			cur.Request.Measure = v.Measure
			workSpaces = append(workSpaces, cur)
		}
	}
	return workSpaces
}

func SingletonWorkSpaces() *WorkSpaces {
	if gWorkSpaces == nil {
		gWorkSpaces = &WorkSpaces{}
	}
	return gWorkSpaces
}
