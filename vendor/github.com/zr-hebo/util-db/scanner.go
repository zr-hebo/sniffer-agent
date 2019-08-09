package db

import (
	"database/sql"
	"fmt"
)

// CheckPair CheckPair
type CheckPair struct {
	Host
	UserErp string `json:"erp"`
	DBName  string `json:"dbname"`
}

func (cp *CheckPair) String() string {
	return fmt.Sprintf(
		"check if %s can visit %s@%s:%d", cp.UserErp, cp.DBName, cp.IP, cp.Port)
}

// Scanner SQL rows读取器
type Scanner interface {
	Scan(*sql.Rows, ...*string) error
}

type showNullScanner struct {
}

// NewShowNullScanner 显示NULL值的接收器，要求Scan的时候传入的参数是string类型
func NewShowNullScanner() (s Scanner) {
	return new(showNullScanner)
}

func (ns *showNullScanner) Scan(
	rows *sql.Rows, receivers ...*string) (err error) {
	nullReceivers := make([]interface{}, 0, len(receivers))
	for range receivers {
		nullReceivers = append(nullReceivers, &sql.NullString{})
	}

	if err = rows.Scan(nullReceivers...); err != nil {
		return
	}

	for i, rv := range nullReceivers {
		nullReceiver := rv.(*sql.NullString)
		if nullReceiver.Valid {
			*receivers[i] = nullReceiver.String
		} else {
			*receivers[i] = "NULL"
		}
	}

	return
}
