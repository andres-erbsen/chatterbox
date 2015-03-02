package main

import (
	"flag"
	"log"

	"github.com/andres-erbsen/chatterbox/client/persistence"
	"github.com/andres-erbsen/chatterbox/proto"
)

var root = flag.String("root", "", "chatterbox root directory")
var subject = flag.String("subject", "", "used to refer to the conversation")

func main() {
	flag.Parse()
	p := &persistence.Paths{
		RootDir:     *root,
		Application: "chat-create",
	}
	if *root == "" || *subject == "" {
		flag.Usage()
		log.Fatal("no root or subject specified")
	}

	metadata := &proto.ConversationMetadata{
		Participants: flag.Args(),
		Subject:      *subject,
	}

	if err := p.ConversationToOutbox(metadata); err != nil {
		log.Fatal(err)
	}
}
