package mysql

import (
	"errors"
	"time"
)

// Command information.
const (
	ComSleep byte = iota
	ComQuit
	ComInitDB
	ComQuery
	ComFieldList
	ComCreateDB
	ComDropDB
	ComRefresh
	ComShutdown
	ComStatistics
	ComProcessInfo
	ComConnect
	ComProcessKill
	ComDebug
	ComPing
	ComTime
	ComDelayedInsert
	ComChangeUser
	ComBinlogDump
	ComTableDump
	ComConnectOut
	ComRegisterSlave
	ComStmtPrepare
	ComStmtExecute
	ComStmtSendLongData
	ComStmtClose
	ComStmtReset
	ComSetOption
	ComStmtFetch
	ComBinlogDumpGtid
	ComResetConnection
)

const (
	maxSQLLen = 5*1024*1024
)

// Client information.
const (
	ClientLongPassword uint32 = 1 << iota
	ClientFoundRows
	ClientLongFlag
	ClientConnectWithDB
	ClientNoSchema
	ClientCompress
	ClientODBC
	ClientLocalFiles
	ClientIgnoreSpace
	ClientProtocol41
	ClientInteractive
	ClientSSL
	ClientIgnoreSigpipe
	ClientTransactions
	ClientReserved
	ClientSecureConnection
	ClientMultiStatements
	ClientMultiResults
	ClientPSMultiResults
	ClientPluginAuth
	ClientConnectAtts
	ClientPluginAuthLenencClientData
)


// Auth name information.
const (
	AuthName = "mysql_native_password"
)


// MySQL database and tables.
const (
	// SystemDB is the name of system database.
	SystemDB = "mysql"
	// UserTable is the table in system db contains user info.
	UserTable = "User"
	// DBTable is the table in system db contains db scope privilege info.
	DBTable = "DB"
	// GlobalVariablesTable is the table contains global system variables.
	GlobalVariablesTable = "GLOBAL_VARIABLES"
	// GlobalStatusTable is the table contains global status variables.
	GlobalStatusTable = "GLOBAL_STATUS"
)


// Identifier length limitations.
// See https://dev.mysql.com/doc/refman/5.7/en/identifiers.html
const (
	// MaxMysqlPacketLen is the max packet payload length.
	MaxMysqlPacketLen = 512 * 1024
)

const (
	millSecondUnit = int64(time.Millisecond)
)

var (
	ErrMalformPacket = errors.New("malform packet error")
)