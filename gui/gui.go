//go:generate genqrc qml

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/andres-erbsen/chatterbox/client/persistence"
	"github.com/andres-erbsen/chatterbox/proto"
	"gopkg.in/qml.v1"
)

var root = flag.String("root", "", "chatterbox root directory")

type gui struct {
	persistence.Paths
	engine *qml.Engine
}

func main() {
	flag.Parse()
	if *root == "" {
		fmt.Fprintf(os.Stderr, "USAGE: %s -root=ROOTDIR", os.Args[0])
		os.Exit(1)
	}

	g := new(gui)
	g.Paths = persistence.Paths{
		RootDir:     *root,
		Application: "chat-create",
	}
	if err := qml.Run(g.run); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}
}

type Conversation struct {
	Subject     string
	Users       []string
	LastMessage string
}

func toJson(v interface{}) string {
	rawJson, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(rawJson)
}

func newConversation(engine *qml.Engine) error {
	controls, err := engine.LoadFile("qml/new-conversation.qml")
	if err != nil {
		return err
	}
	window := controls.CreateWindow(nil)

	window.On("sendMessage", func(to, subject, message string) {
		println("To: " + to)
		println("Subject: " + subject)
		println("Message: " + message)
	})

	return nil
}

func (g *gui) conversation(idx int) error {
	println(idx)
	controls, err := g.engine.LoadFile("qml/old-conversation.qml")
	if err != nil {
		return err
	}
	window := controls.CreateWindow(nil)
	messageModel := window.ObjectByName("messageModel")

	msgs, err := g.LoadMessages(&proto.ConversationMetadata{Subject: "thread", Participants: []string{"andres", "andreser"}})
	if err != nil {
		panic(err)
	}
	for _, msg := range msgs {
		msg.Content = strings.TrimSpace(msg.Content)
		messageModel.Call("addItem", toJson(msg))
	}
	window.ObjectByName("messageView").Call("positionViewAtEnd")

	window.On("sendMessage", func(message string) {
		println("Send: " + message)
	})

	return nil
}

func (g *gui) run() error {
	g.engine = qml.NewEngine()
	controls, err := g.engine.LoadFile("qml/history.qml")
	if err != nil {
		return err
	}

	window := controls.CreateWindow(nil)

	listModel := window.ObjectByName("listModel")
	convs, err := g.ListConversations()
	if err != nil {
		return err
	}
	for _, con := range convs {
		c := Conversation{Subject: con.Subject, Users: con.Participants}
		listModel.Call("addItem", toJson(c))
	}

	window.ObjectByName("table").On("clicked", g.conversation)

	window.Show()
	window.Wait()
	return nil
}
