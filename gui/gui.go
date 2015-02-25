//go:generate genqrc qml

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/andres-erbsen/chatterbox/client/persistence"
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

type Message struct {
	Sender  string
	Content string
}

func (con *Conversation) toJson() string {
	raw_json, err := json.Marshal(con)
	if err != nil {
		panic(err)
	}
	return string(raw_json)
}

func (msg *Message) toJson() string {
	raw_json, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return string(raw_json)
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

func oldConversation(engine *qml.Engine) error {
	controls, err := engine.LoadFile("qml/old-conversation.qml")
	if err != nil {
		return err
	}
	window := controls.CreateWindow(nil)

	messages := []Message{Message{Sender: "Bill", Content: "test 1"}}

	me := "Bob"

	messageModel := window.ObjectByName("messageModel")
	for _, message := range messages {
		raw_json, _ := json.Marshal(message)
		messageModel.Call("addItem", string(raw_json))
	}
	window.ObjectByName("messageView").Call("positionViewAtEnd")

	window.On("sendMessage", func(message string) {
		//println("To: " + to)
		//println("Subject: " + subject)
		raw_json, _ := json.Marshal(Message{Sender: me, Content: message})
		messageModel.Call("addItem", string(raw_json))
		window.ObjectByName("messageView").Call("positionViewAtEnd")
		println("Message: " + message)
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
		listModel.Call("addItem", c.toJson())
	}

	//window.ObjectByName("table").On("clicked", func() {newConversation(g.engine)})
	window.ObjectByName("table").On("clicked", g.conversation())

	window.Show()
	window.Wait()
	return nil
}
