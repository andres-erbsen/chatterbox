package daemon

import (
	"github.com/andres-erbsen/chatterbox/proto"
	"time"
)

type Config struct {
	// The root directory where the user's files are stored
	RootDir string

	// Gets the current time
	Now func() time.Time

	// Prefix used in the temp folder
	TempPrefix string

	proto.LocalAccountConfig
}

func LoadConfig(conf *Config) *Config {
	UnmarshalFromFile(conf.ConfigFile(), &conf.LocalAccountConfig)
	return conf
}
