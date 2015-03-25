package daemon

import (
	"fmt"
	"net"
	"testing"

	protobuf "code.google.com/p/gogoprotobuf/proto"
	cbClient "github.com/andres-erbsen/chatterbox/client"
	"github.com/andres-erbsen/chatterbox/client/persistence"
	"github.com/andres-erbsen/chatterbox/proto"
	denameClient "github.com/andres-erbsen/dename/client"
	testutil2 "github.com/andres-erbsen/dename/server/testutil" //TODO: Move MakeToken to TestUtil
)

func PrepareTestAccountDaemon(name string, rootDir string, denameConfig *denameClient.Config, serverAddr string, serverPk *[32]byte, t testing.TB) *Daemon {
	dnmClient, err := denameClient.NewClient(denameConfig, nil, nil)

	//get port from serverAddr
	addr, portStr, err := net.SplitHostPort(serverAddr)
	if err != nil {
		t.Fatal(err)
	}
	var port int
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
		t.Fatal(err)
	}

	//create the accounts with Init
	torAddr := "DANGEROUS_NO_TOR"
	err = Init(rootDir, name, addr, port, serverPk, torAddr)
	if err != nil {
		t.Fatal(err)
	}

	//initialize daemon with Load
	theDaemon, err := Load(rootDir, torAddr, denameConfig)
	if err != nil {
		t.Fatal(err)
	}

	cbProfile := new(proto.Profile)
	if err := persistence.UnmarshalFromFile(theDaemon.OurChatterboxProfilePath(), cbProfile); err != nil {
		t.Fatal(err)
	}

	chatProfileBytes, err := protobuf.Marshal(cbProfile)
	if err != nil {
		t.Fatal(err)
	}

	denameProfile, sk, err := denameClient.NewProfile(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := denameClient.SetProfileField(denameProfile, cbClient.PROFILE_FIELD_ID, chatProfileBytes); err != nil {
		t.Fatal(err)
	}

	err = dnmClient.Register(sk, name, denameProfile, testutil2.MakeToken())
	if err != nil {
		t.Fatal(err)
	}

	return theDaemon
}
