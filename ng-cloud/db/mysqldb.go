package db

import (
	"fmt"
	"time"

	log4plus "github.com/nextGPU/include/log4go"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type MysqlManager struct {
	connected       bool
	MysqlIp         string
	MysqlPort       int
	MysqlDBName     string
	MysqlDBCharset  string
	UserName        string
	Password        string
	disconnectTimer int64
	Mysqldb         *sqlx.DB
}

func (m *MysqlManager) connectMysql() bool {
	funName := "connectMysql"
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&multiStatements=true", m.UserName, m.Password, m.MysqlIp, m.MysqlPort, m.MysqlDBName, m.MysqlDBCharset)
	if m.Mysqldb == nil {
		log4plus.Info("🔗 Mysqldb dsn=%s", dsn)
		var err error
		if m.Mysqldb, err = sqlx.Connect("mysql", dsn); err != nil {
			log4plus.Error("%s mySQL Connect Failed dsn=[%s] err=[%s]", funName, dsn, err.Error())
			m.connected = false
			return false
		}
		m.Mysqldb.SetMaxOpenConns(20)
		m.Mysqldb.SetMaxIdleConns(10)
		m.Mysqldb.SetConnMaxLifetime(time.Hour)
		rows, err := m.Mysqldb.Query("select * from `t_nodes` limit 1;")
		if err != nil {
			m.connected = false
			m.disconnectTimer = time.Now().Unix()
			log4plus.Error("Check Mysql Connect Failed err=%s", err.Error())
		} else {
			rows.Close()
			m.connected = true
			log4plus.Info("connectMysql connected is ture")
		}
	} else {
		log4plus.Info("connectMysql Mysqldb is nil dsn=%s", dsn)
		if m.connected {
			if err := m.Mysqldb.Ping(); err != nil {
				m.connected = false
				return false
			}
		} else {
			log4plus.Info("connectMysql Mysqldb Close->Open")
			var err error
			m.Mysqldb.Close()
			if m.Mysqldb, err = sqlx.Open("mysql", dsn); err != nil {
				log4plus.Error("%s mySQL Connect Failed dsn=%s", funName, dsn)
				m.connected = false
				return false
			}
			if err = m.Mysqldb.Ping(); err != nil {
				m.connected = false
				return false
			}
			m.connected = true
		}
	}
	return m.connected
}

func (m *MysqlManager) IsConnect() bool {
	return m.connected
}

func (m *MysqlManager) pollMysql() {
	funName := "pollMysql"
	for {
		if m.Mysqldb != nil {
			rows, err := m.Mysqldb.Query("select * from `t_nodes` limit 1;")
			if err != nil {
				m.connected = false
				m.disconnectTimer = time.Now().Unix()
				log4plus.Error("%s Check Mysql Connect Failed err=%s", funName, err.Error())
			} else {
				rows.Close()
			}
		} else {
			m.connected = m.connectMysql()
			if !m.connected {
				m.disconnectTimer = time.Now().Unix()
				log4plus.Error("%s ReConnect Mysql Failed", funName)
			}
		}
		if !m.IsConnect() {
			if time.Now().Unix()-m.disconnectTimer >= 300 {
				m.connected = m.connectMysql()
				if !m.connected {
					m.disconnectTimer = time.Now().Unix()
					log4plus.Error("%s ReConnect Mysql Failed", funName)
				}
			}
		}
		time.Sleep(30 * time.Second)
	}

}

func NewMysql(Ip string, Port int, DBName string, DBCharset string, UserName string, Password string) *MysqlManager {
	sql := &MysqlManager{
		MysqlIp:        Ip,
		MysqlPort:      Port,
		MysqlDBName:    DBName,
		MysqlDBCharset: DBCharset,
		UserName:       UserName,
		Password:       Password,
		connected:      false,
	}
	go sql.pollMysql()
	return sql
}
