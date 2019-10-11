package db

import (
	"database/sql"
	"sync"
	"time"
)

// PooledMysqlDB Mysql主机实例
type PooledMysqlDB struct {
	MysqlDB
	conn *sql.DB
	lock *sync.Mutex
}

// NewPooledMysqlDBWithParam 带参数创建MySQL数据库
func NewPooledMysqlDBWithParam(
	ip string, port int, userName, passwd string) (
	pmd *PooledMysqlDB) {
	pmd = NewPooledMysqlDB()
	pmd.IP = ip
	pmd.Port = port
	pmd.UserName = userName
	pmd.Passwd = passwd
	pmd.DatabaseType = dbTypeMysql

	return
}

// NewPooledMysqlDBWithAllParam 带参数创建MySQL数据库
func NewPooledMysqlDBWithAllParam(
	ip string, port int, userName, passwd, dbName string) (
	pmd *PooledMysqlDB) {
	pmd = NewPooledMysqlDB()
	pmd.IP = ip
	pmd.Port = port
	pmd.UserName = userName
	pmd.Passwd = passwd
	pmd.DBName = dbName

	return
}

// NewPooledMysqlDB 创建MySQL数据库
func NewPooledMysqlDB() (pmd *PooledMysqlDB) {
	pmd = new(PooledMysqlDB)
	pmd.DatabaseType = dbTypeMysql
	pmd.QueryTimeout = 5
	pmd.lock = new(sync.Mutex)
	return
}

// CloseConnection 获取数据库连接
func (pmd *PooledMysqlDB) CloseConnection() (err error) {
	if pmd.conn == nil {
		return
	}

	err = pmd.conn.Close()
	return
}

// GetConnection 获取数据库连接
func (pmd *PooledMysqlDB) GetConnection() (conn *sql.DB, err error) {
	pmd.lock.Lock()
	defer func() {
		pmd.lock.Unlock()
	}()

	if pmd.conn != nil {
		conn = pmd.conn
		return
	}

	conn, err = pmd.MysqlDB.getConnection()
	if err != nil {
		return
	}

	conn.SetConnMaxLifetime(time.Second * 60 * 30)
	conn.SetMaxOpenConns(0)
	if err := conn.Ping(); err != nil {
		return nil, err
	}
	pmd.conn = conn

	return
}

// ExecChange 执行MySQL Query语句
func (pmd *PooledMysqlDB) ExecChange(stmt string, args ...interface{}) (
	result sql.Result, err error) {
	return pmd.MysqlDB.ExecChange(stmt, args...)
}

// QueryRows 执行MySQL Query语句
func (pmd *PooledMysqlDB) QueryRows(stmt string) (queryRows *QueryRows, err error) {
	return pmd.MysqlDB.QueryRows(stmt)

}

// QueryRow 执行MySQL Query语句
func (pmd *PooledMysqlDB) QueryRow(stmt string) (row *QueryRow, err error) {
	return pmd.MysqlDB.QueryRow(stmt)
}
