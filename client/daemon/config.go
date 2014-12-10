package daemon

import (
	"time"
)

type Config struct {
	// The root directory where the user's files are stored
	RootDir string

	// Gets the current time
	Now func() time.Time

	// Prefix used in the temp folder
	TempPrefix string
}
