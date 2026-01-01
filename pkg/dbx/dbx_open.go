// If we have Darwin problems,
// make a version with no ODBC and build
// tag it to Darwin

package dbx

import (
	"database/sql"

	_ "github.com/alexbrainman/odbc"
	_ "github.com/microsoft/go-mssqldb"
	"github.com/pkg/errors"
)

// Open is a platform specific open command
func Open(driver, cxnstr string) (*sql.DB, error) {
	db, err := sql.Open(driver, cxnstr)
	if err != nil {
		if db != nil {
			db.Close()
		}
		return db, errors.Wrap(err, "sql.open")
	}
	return db, nil
}
