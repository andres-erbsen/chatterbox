package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/andres-erbsen/chatterbox/client/daemon"
)

type hex32Byte [32]byte

func (h *hex32Byte) String() string {
	return hex.EncodeToString(h[:])
}
func (h *hex32Byte) Set(value string) error {
	if len(value) != 2*32 {
		return fmt.Errorf("Server pubkey must be 64 hex digits long, got %d", len(value))
	}
	_, err := hex.Decode(h[:], []byte(value))
	if err != nil {
		return err
	}
	return err
}

func main() {
	dename := flag.String("dename", "", "Your dename username.")
	var serverTransportPubkey [32]byte
	if err := ((*hex32Byte)(&serverTransportPubkey)).Set("70eb7fb3e6c85c006ba76b48208ccf75f99eb6ec98dffb4292388369cb197e30"); err != nil {
		panic(err)
	}
	flag.Var((*hex32Byte)(&serverTransportPubkey), "server-pubkey", "The TCP port which the server listens on. Note that people sending you mesages expct to be able to reach your home server at port 1984.")
	serverAddress := flag.String("server-host", "chatterbox.xvm.mit.edu", "The IP address or hostname on which your (prospective) home server server can be reached")
	serverPort := flag.Int("server-port", 1984, "The TCP port which the server listens on.")
	dir := flag.String("account-directory", "", "Dedicated directory for the account.")
	torAddress := flag.Bool("tor-address", "127.0.0.1:9050", "Address of the local TOR proxy.")
	flag.Parse()

	if *dename == "" || serverTransportPubkey == [32]byte{} || *serverAddress == "" {
		flag.Usage()
		os.Exit(2)
	}
	if *dir == "" {
		*dir = filepath.Join(os.Getenv("HOME"), ".chatterbox", *dename)
	}

	if err := daemon.Init(*dir, *dename, *serverAddress, *serverPort, &serverTransportPubkey, *torAddress); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Account initialization done.\n"+
		"You may use the following command to link this account with your dename profile:.\n"+
		"torify dnmgr set '%s' 1984 < %s/.daemon/chatterbox-profile.pb\n", *dename, *dir)
}
