package mysql

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	du "github.com/zr-hebo/util-db"
)

func expandLocalMysql(port int) (mysqlHost *du.MysqlDB) {
	mysqlHost = new(du.MysqlDB)
	mysqlHost.IP = "localhost"
	mysqlHost.Port = port
	mysqlHost.UserName = adminUser
	mysqlHost.Passwd = adminPasswd
	mysqlHost.DatabaseType = "mysql"
	mysqlHost.ConnectTimeout = 1

	return
}

func querySessionInfo(snifferPort int, clientHost *string) (user, db *string, err error) {
	mysqlServer := expandLocalMysql(snifferPort)
	querySQL := fmt.Sprintf(
		"SELECT user, db FROM information_schema.processlist WHERE host='%s'", clientHost)
	// log.Debug(querySQL)
	queryRow, err := mysqlServer.QueryRow(querySQL)
	if err != nil {
		return
	}

	if queryRow == nil {
		return
	}

	userVal := queryRow.Record["user"]
	if userVal != nil {
		usrStr := userVal.(string)
		user = &usrStr
	}

	dbVal := queryRow.Record["db"]
	if dbVal != nil {
		dbStr := dbVal.(string)
		db = &dbStr
	}

	return
}