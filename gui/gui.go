//go:generate genqrc qml

package main

import (
	"fmt"
	"gopkg.in/qml.v1"
	"encoding/json"
	"os"
)

func main() {
	if err := qml.Run(run); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

type Conversation struct {
	Subject string
	Users []string
	LastMessage string
}


type Message struct {
	Sender string
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

	messages := []Message{ Message{Sender:"Bill",Content:"test 1"} }

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
		raw_json, _ := json.Marshal(Message{Sender:me,Content:message})
		messageModel.Call("addItem", string(raw_json))
		window.ObjectByName("messageView").Call("positionViewAtEnd")
		println("Message: " + message)
	})

	return nil
}

func run() error {

	engine := qml.NewEngine()

	controls, err := engine.LoadFile("qml/history.qml")
	if err != nil {
		return err
	}

	con1 := Conversation{Subject:"subject", Users:[]string{"Bob", "Jane"}, LastMessage:"hello?"}
	con2 := Conversation{Subject:"elephants", Users:[]string{"Bob"}, LastMessage:"I forgot what I was going to say."}
	con3 := Conversation{Subject:"tigers", Users:[]string{"Jane"}, LastMessage:"oh my"}

	history := []Conversation{con1, con2, con3}

	window := controls.CreateWindow(nil)


	listModel := window.ObjectByName("listModel")
	for _, con := range history {
		listModel.Call("addItem", con.toJson())
	}

	//window.ObjectByName("table").On("clicked", func() {newConversation(engine)})
	window.ObjectByName("table").On("clicked", func() {oldConversation(engine)})

	window.Show()
	window.Wait()
	return nil
}
