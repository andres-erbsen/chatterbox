package main

import (
	"flag"
	"github.com/andres-erbsen/chatterbox/proto"
	//	"github.com/andres-erbsen/client/clientutil"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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
	var noPubkey hex32Byte
	var serverTransportPubkey hex32Byte
	flag.Var(&serverTransportPubkey, "server-pubkey", "The TCP port which the server listens on. Note that people sending you mesages expct to be able to reach your home server at port 1984.")
	serverAddress := flag.String("server-host", "", "The IP address or hostname on which your (prospective) home server server can be reached")
	serverPort := flag.Int("server-port", 1984, "The TCP port which the server listens on. Note that people sending you mesages expct to be able to reach your home server at port 1984.")
	dir := flag.String("account-directory", "", "Dedicated directory for the account.")
	flag.Parse()

	if *dename == "" || serverTransportPubkey == noPubkey || *serverAddress == "" || serverPort == nil {
		flag.Usage()
		os.Exit(2)
	}
	if *dir == "" {
		*dir = filepath.Join(os.Getenv("HOME"), ".chatterbox", *dename)
	}

	secretConfig := &proto.LocalAccountConfig{
		ServerAddressTCP: *serverAddress,
		ServerPortTCP:    int32(*serverPort),
	}
	publicProfile := &proto.Profile{
		ServerAddressTCP: *serverAddress,
	}
	/* TODO: enable
	if err := clientutil.GenerateLongTermKeys(secretConfig, publicProfile); err != nil {
		panic(err)
	}
	*/
	secretConfigBytes, err := secretConfig.Marshal()
	if err != nil {
		panic(err)
	}
	publicProfileBytes, err := publicProfile.Marshal()
	if err != nil {
		panic(err)
	}

	if err := os.Mkdir(*dir, 0700); err != nil && !os.IsExist(err) {
		fmt.Fprintf(os.Stderr, "could not create directory %s: %s", *dir, err)
		os.Exit(2)
	}
	configFilePath := filepath.Join(*dir, "config.pb")
	if _, err := os.Stat(configFilePath); !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "file already exists: %s\n", configFilePath)
		os.Exit(2)
	}
	if err := ioutil.WriteFile(configFilePath, secretConfigBytes, 0600); err != nil {
		// TODO: "WriteFileSync" -- issue fsync after write
		fmt.Fprintf(os.Stderr, "error writing file %s: %s\n", err)
		os.Exit(2)
	}

	// TODO: when should we create the account the at the server?

	fmt.Printf("Local account initialization done.\n")
	fmt.Printf("You may use the following command to link this account with your dename profile:.\n"+
		"echo -n %x | xxd -r -p | dnmgr set '%s' 1984 -\n", publicProfileBytes, *dename)
}
