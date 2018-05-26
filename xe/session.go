package xe

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

var (
	// ErrNotFound is returnd non-existant session
	ErrNotFound = errors.New("session not found")
	// ErrNotRunning is return for autostart sessions that aren't running
	ErrNotRunning = errors.New("session not running")
	// ErrNofileTarget is for running sessions that have no file target
	ErrNofileTarget = errors.New("no file target")
	// ErrInvalidState is for any other situation
)

// if isrunning and hasfiletarget then OK
// if no rows (session doesn't exist), warning "ErrNotFound" -- ignore unless strict (TODO)
// if not autostart and is always on, then OK
// if autostart not isrunning then warning "ErrNotRunning" -- error 
// if isrunning and not hasfiletarget then warning "ErrNoFileTarget" -- error

// ValidateSession confirms that a session is valid and has a file target
func ValidateSession(db *sql.DB, session string) error {
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
		return ErrNofileTarget
	}
	if !autostart && !isrunning {
		return nil
	}
	return fmt.Errorf("autostart: %v; running: %v filetarget: %v", autostart, isrunning, hasfiletarget)
}
