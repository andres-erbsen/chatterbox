// client daemon
//   watches the file system for new messages --> sends them
//   communicates with the server --> receive new messages
package main

import (
	"github.com/andres-erbsen/chatterbox/client/util/filesystem"
)

func main() {
	rootDir := filesystem.GetRootDir()
	filesystem.InitFs(rootDir)
}
