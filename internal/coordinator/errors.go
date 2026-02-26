package coordinator

import "errors"

var (
	// ErrTimestampDirExists is returned when a timestamp directory already exists
	ErrTimestampDirExists = errors.New("timestamp directory already exists")
	
	// ErrTimestampNotFound is returned when a specified timestamp directory does not exist
	ErrTimestampNotFound = errors.New("timestamp directory not found")
)
