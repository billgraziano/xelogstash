package xe

import (
	"database/sql"

	"github.com/pkg/errors"
)

// ErrInvalidSession is returnd for invalid, stopped or non-file target sessions
var ErrInvalidSession = errors.New("invalid session or no file target or session stopped")

// ValidateSession confirms that a session is valid and has a file target
func ValidateSession(db *sql.DB, session string) error {
	query := `SELECT S.[name]
	FROM	sys.dm_xe_sessions S 
	JOIN	sys.dm_xe_session_targets T ON T.event_session_address = S.[address]
	WHERE	target_name = 'event_file'
	AND 	S.[name] = ?`

	var name string
	err := db.QueryRow(query, session).Scan(&name)
	switch {
	case err == sql.ErrNoRows:
		return ErrInvalidSession
	case err != nil:
		return errors.Wrap(err, "query")
	default:
	}
	return nil
}
