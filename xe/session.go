package xe

import (
	"database/sql"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

var (
	// ErrNotFound is returnd non-existant session
	ErrNotFound = errors.New("session not found")
	// ErrNotRunning is return for autostart sessions that aren't running
	ErrNotRunning = errors.New("session not running")
	// ErrNoFileTarget is for running sessions that have no file target
	ErrNoFileTarget = errors.New("no file target")
	// ErrInvalidState is for any other situation
)

// Session holds the XE session from SQL Server.
// Mainly the file name and the wildcard
type Session struct {
	Name     string
	Filename string
	WildCard string
}

// GetSession returns an XE session from the database
func GetSession(db *sql.DB, session string) (s Session, err error) {
	query := `
	
		SELECT 
			ses.[name],
			--sesf.[event_session_id],
			--sesf.[object_id],
			-- sesf.[name],
			CAST(sesf.[value] AS NVARCHAR(1024)) AS [value]
		FROM sys.server_event_sessions ses 
		JOIN sys.server_event_session_targets sest ON sest.event_session_id = ses.event_session_id
		JOIN sys.server_event_session_fields sesf ON sesf.event_session_id = ses.event_session_id
												AND sesf.object_id = sest.target_id 
		WHERE	sesf.[name] = 'filename'
		AND		ses.[name] = ?
	`
	err = db.QueryRow(query, session).Scan(&s.Name, &s.Filename)
	if err != nil {
		return s, errors.Wrap(err, "db.queryrow.scan")
	}

	// if no extension, add one
	ext := filepath.Ext(s.Filename)
	if ext == "" {
		s.Filename += ".xel"
	}

	// build the wildcard
	basePath := strings.TrimSuffix(s.Filename, path.Ext(s.Filename))
	s.WildCard = basePath + "*" + path.Ext(s.Filename)

	return s, err
}

// ValidateSession confirms that a session is valid and has a file target
func ValidateSession(db *sql.DB, session string) error {

	// if isrunning and hasfiletarget then OK
	// if no rows (session doesn't exist), warning "ErrNotFound" -- ignore unless strict (TODO)
	// if not autostart and is always on, then OK
	// if autostart not isrunning then warning "ErrNotRunning" -- error
	// if isrunning and not hasfiletarget then warning "ErrNoFileTarget" -- error

	query := `
		SELECT	[session].[name],
			[session].[startup_state] AS [AutoStart],
			--[running].[create_time] AS [StartTime],
			CAST((CASE WHEN ([running].[create_time] IS NULL) THEN 0 ELSE 1 END) AS BIT)AS [IsRunning],
			CAST((CASE WHEN T.[event_session_address] IS NOT NULL THEN 1 ELSE 0 END) AS BIT) AS [HasFileTarget]
		FROM	[sys].[server_event_sessions] AS [session]
		LEFT OUTER JOIN 
			[sys].[dm_xe_sessions] AS [running] ON [running].[name] = [session].[name]
		LEFT OUTER JOIN 
		[sys].[dm_xe_session_targets] T ON T.[event_session_address] = [running].[address]
						AND T.[target_name] = 'event_file'
		WHERE 	[session].[name] = ?`

	var name string
	var autostart, isrunning, hasfiletarget bool
	err := db.QueryRow(query, session).Scan(&name, &autostart, &isrunning, &hasfiletarget)
	if err == nil && isrunning && hasfiletarget {
		return nil
	}
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if autostart && !isrunning {
		return ErrNotRunning
	}
	if name == "AlwaysOn_health" && !autostart {
		return nil
	}
	if isrunning && !hasfiletarget {
		return ErrNoFileTarget
	}
	if !autostart && !isrunning {
		return nil
	}
	return fmt.Errorf("autostart: %v; running: %v filetarget: %v", autostart, isrunning, hasfiletarget)
}
