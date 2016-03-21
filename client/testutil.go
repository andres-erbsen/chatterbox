package client

import (
	"crypto/rand"
	"fmt"
	"net"
	"testing"

	protobuf "golang.org/x/oprotobuf/proto"
	"github.com/andres-erbsen/chatterbox/proto"
	"github.com/andres-erbsen/chatterbox/transport"
	"github.com/andres-erbsen/dename/client"
	testutil2 "github.com/andres-erbsen/dename/server/testutil" //TODO: Move MakeToken to TestUtil
)

func CreateTestDenameAccount(name string, denameClient *client.Client, secretConfig *proto.LocalAccountConfig, serverAddr string, serverPk *[32]byte, t testing.TB) {
	//TODO: move this to test?
	//TODO: All these names are horrible, please change them
	addr, portStr, err := net.SplitHostPort(serverAddr)
	if err != nil {
		t.Fatal(err)
	}
	var port int32
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
		t.Fatal(err)
	}

	chatProfile := &proto.Profile{
		ServerAddressTCP:  addr,
		ServerPortTCP:     port,
		ServerTransportPK: (proto.Byte32)(*serverPk),
	}

	if err := GenerateLongTermKeys(secretConfig, chatProfile, rand.Reader); err != nil {
		t.Fatal(err)
	}

	chatProfileBytes, err := protobuf.Marshal(chatProfile)
	if err != nil {
		t.Fatal(err)
	}

	profile, sk, err := client.NewProfile(nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	client.SetProfileField(profile, PROFILE_FIELD_ID, chatProfileBytes)

	err = denameClient.Register(sk, name, profile, testutil2.MakeToken())
	if err != nil {
		t.Fatal(err)
	}
}

func CreateTestAccount(name string, denameClient *client.Client, secretConfig *proto.LocalAccountConfig, serverAddr string, serverPk *[32]byte, t testing.TB) *transport.Conn {

	CreateTestDenameAccount(name, denameClient, secretConfig, serverAddr, serverPk, t)
	conn := CreateTestHomeServerConn(name, denameClient, secretConfig, t)

	inBuf := make([]byte, proto.SERVER_MESSAGE_SIZE)

	err := CreateAccount(conn, inBuf)
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func CreateTestHomeServerConn(dename string, denameClient *client.Client, secretConfig *proto.LocalAccountConfig, t testing.TB) *transport.Conn {
	profile, err := denameClient.Lookup(dename)
	if err != nil {
		t.Fatal(err)
	}

	chatProfileBytes, err := client.GetProfileField(profile, PROFILE_FIELD_ID)
	if err != nil {
		t.Fatal(err)
	}

	chatProfile := new(proto.Profile)
	if err := chatProfile.Unmarshal(chatProfileBytes); err != nil {
		t.Fatal(err)
	}

	addr := chatProfile.ServerAddressTCP
	port := chatProfile.ServerPortTCP
	pkTransport := ([32]byte)(chatProfile.ServerTransportPK)
	pkp := (*[32]byte)(&chatProfile.UserIDAtServer)

	oldConn, err := net.Dial("tcp", net.JoinHostPort(addr, fmt.Sprint(port)))
	if err != nil {
		t.Fatal(err)
	}

	skp := (*[32]byte)(&secretConfig.TransportSecretKeyForServer)

	conn, _, err := transport.Handshake(oldConn, pkp, skp, &pkTransport, proto.SERVER_MESSAGE_SIZE)
	if err != nil {
		t.Fatal(err)
	}

	return conn
}
