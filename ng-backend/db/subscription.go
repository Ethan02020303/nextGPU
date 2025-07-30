package db

import (
	"errors"
	"fmt"
	log4plus "github.com/nextGPU/include/log4go"
	"github.com/nextGPU/ng-backend/configure"
	"time"
)

type SubscriptionDB struct {
	mysqlDb *MysqlManager
}

var gSubscriptionDB *SubscriptionDB

func (p *SubscriptionDB) InsertSubscription(orderID, userName, subscriptionID string, amount float64, mode int64) error {
	funName := "InsertSubscription"
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s orderID=[%s] userName=[%s] subscriptionID=[%s] amount=[%d] mode=[%d] consumption time=%d(ms)",
			funName, orderID, userName, subscriptionID, amount, mode, time.Now().UnixMilli()-now)
	}()

	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}

	sql := fmt.Sprintf(`insert into t_subscription (f_order_id, f_user_name, f_amount, f_subscription_id, f_mode, f_create_time, f_state) values 
                                                    ('%s', '%s', %f, '%s', %d, NOW(), 'Scaning');`,
		orderID, userName, amount, subscriptionID, mode)
	_, err := p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s insert Failed err=[%s] orderID=[%s] userName=[%s] subscriptionID=[%s] amount=[%.2f] mode=[%d]",
			funName, err.Error(), orderID, userName, subscriptionID, amount, mode)
		return err
	}
	return nil
}

func (p *SubscriptionDB) UpdateSubscription(orderID, state string, amount int) error {
	funName := "UpdateSubscription"
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s orderID=[%s] state=[%s] amont=[%d] consumption time=%d(ms)",
			funName, orderID, state, amount, time.Now().UnixMilli()-now)
	}()

	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`update t_subscription set f_state='%s', f_amount=%d, f_completion_time=NOW() where f_order_id='%s';`, state, amount, orderID)
	_, err := p.mysqlDb.Mysqldb.Exec(sql)
	if err != nil {
		log4plus.Error("%s insert Failed err=[%s] state=[%s] f_amount=[%d]", funName, err.Error(), orderID, state, amount)
		return err
	}
	return nil
}

func (p *SubscriptionDB) CheckOrderStatus(orderID string) (error, string) {
	funName := "CheckOrderStatus"
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s orderID=[%s]consumption time=%d(ms)", funName, orderID, time.Now().UnixMilli()-now)
	}()
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString), ""
	}
	sql := fmt.Sprintf(`select IFNULL(f_id, -1), IFNULL(f_state, '') from t_subscription where f_order_id='%s';`, orderID)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return errors.New(errString), ""
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var status string
		scanErr := rows.Scan(&id, &status)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if id == -1 {
			return errors.New("not found orderID"), "FAILED"
		} else {
			return nil, status
		}
	}
	return errors.New(""), "FAILED"
}

func (p *SubscriptionDB) getLevelScore(level int) (error, int) {
	funName := "getLevelScore"
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s level=[%d] consumption time=%d(ms)", funName, level, time.Now().UnixMilli()-now)
	}()
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString), 0
	}
	sql := fmt.Sprintf(`select IFNULL(f_id, -1), IFNULL(f_score, 0) from t_levels where f_level=%d;`, level)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return errors.New(errString), 0
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var score int
		scanErr := rows.Scan(&id, &score)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if id == -1 {
			return errors.New("not found orderID"), 0
		} else {
			return nil, score
		}
	}
	return errors.New(""), 0
}

func (p *SubscriptionDB) SetUserVip(orderID, state string) error {
	funName := "SetUserVip"
	now := time.Now().UnixMilli()
	defer func() {
		log4plus.Info("%s orderID=[%s] state=[%s] consumption time=%d(ms)", funName, orderID, state, time.Now().UnixMilli()-now)
	}()
	if !p.mysqlDb.IsConnect() {
		errString := fmt.Sprintf("%s Db Not Connect", funName)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	sql := fmt.Sprintf(`select IFNULL(f_id, -1), IFNULL(f_user_name, ''), IFNULL(f_state, ''),
       IFNULL(f_subscription_id, '') from t_subscription where f_order_id='%s';`, orderID)
	rows, err := p.mysqlDb.Mysqldb.Query(sql)
	if err != nil {
		errString := fmt.Sprintf("%s Query Failed Error=[%s] SQL=[%s]", funName, err.Error(), sql)
		log4plus.Error(errString)
		return errors.New(errString)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var userName, status, subscriptionID string
		scanErr := rows.Scan(&id, &userName, &status, &subscriptionID)
		if scanErr != nil {
			log4plus.Error("%s Scan Error=[%s]", funName, scanErr.Error())
			continue
		}
		if id != -1 {
			var level int
			if subscriptionID == "primary" {
				level = 1
			} else if subscriptionID == "intermediate" {
				level = 2
			} else if subscriptionID == "premium" {
				level = 3
			}
			errScore, score := p.getLevelScore(level)
			if errScore != nil {
				log4plus.Error("%s getLevelScore Failed err=[%s] userName=[%s]", funName, errScore.Error(), userName)
				return err
			}
			sql = fmt.Sprintf(`update t_users set f_level=%d, f_cost=%d, f_vip_start_time=NOW(), 
                   f_vip_end_time=DATE_ADD(NOW(), INTERVAL 30 DAY) where f_user_name='%s';`, level, score, userName)
			_, errUserName := p.mysqlDb.Mysqldb.Exec(sql)
			if errUserName != nil {
				log4plus.Error("%s Update Failed err=[%s] userName=[%s]", funName, errUserName.Error(), userName)
				return err
			}
			return nil
		} else {
			return errors.New("not found userName")
		}
	}
	return errors.New("")
}

func SingletonSubscriptionDB() *SubscriptionDB {
	if gSubscriptionDB == nil {
		log4plus.Info("SingletonSubscriptionDB ---->>>>")
		gSubscriptionDB = &SubscriptionDB{}
		if gSubscriptionDB.mysqlDb = NewMysql(configure.SingletonConfigure().Mysql.MysqlIp,
			configure.SingletonConfigure().Mysql.MysqlPort,
			configure.SingletonConfigure().Mysql.MysqlDBName,
			configure.SingletonConfigure().Mysql.MysqlDBCharset,
			configure.SingletonConfigure().Mysql.UserName,
			configure.SingletonConfigure().Mysql.Password); gSubscriptionDB.mysqlDb == nil {
			log4plus.Error("SingletonSubscriptionDB NewMysql Failed")
			return nil
		}
	}
	return gSubscriptionDB
}
