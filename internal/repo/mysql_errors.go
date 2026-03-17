package repo

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

func isMysqlError(err error) bool {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return me.Number == 1062
	}
	return false
}
