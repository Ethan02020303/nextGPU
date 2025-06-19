package process

import (
	"fmt"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/ng-cloud/db"
)

type Consumptions struct {
}

var gConsumptions *Consumptions

func (w *Consumptions) CalcEstimates(title, systemUUID string, imageWidth, imageHeight int64) (err error, completionTime int64, familyWeight float64) {
	funName := "CalcEstimates"
	log4plus.Info("%s title=[%s] imageWidth=[%d] imageHeight=[%d]", funName, title, imageWidth, imageHeight)
	node := SingletonNodes().FindNode(systemUUID)
	if node == nil {
		errString := fmt.Sprintf("%s FindNode Failed systemUUID=[%s] ", funName, systemUUID)
		log4plus.Error(errString)
		return err, 0, 0
	}
	err, familyID, _, familyWeight := SingletonNodes().GPUFamilyIDVideoRam(node.Base.GPU)
	if err != nil {
		errString := fmt.Sprintf("%s GPUFamilyIDVideoRam Failed systemUUID=[%s] ", funName, systemUUID)
		log4plus.Error(errString)
		return err, 0, 0
	}
	errCompletion, completionTime := db.SingletonNodeBaseDB().GetCompletion(familyID, title, imageWidth, imageHeight)
	if errCompletion != nil {
		errString := fmt.Sprintf("%s GetCompletion Failed familyID=[%d] title=[%s] systemUUID=[%s] ", funName, familyID, title, systemUUID)
		log4plus.Error(errString)
		return errCompletion, 0, 0
	}
	return nil, completionTime, familyWeight
}

func SingletonConsumptions() *Consumptions {
	if gConsumptions == nil {
		gConsumptions = &Consumptions{}
	}
	return gConsumptions
}
