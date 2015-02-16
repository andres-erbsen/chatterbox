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

func (con *Conversation) toJson() string {
	raw_json, _ := json.Marshal(con)
	return string(raw_json)
}

func run() error {

	engine := qml.NewEngine()

	controls, err := engine.LoadFile("qrc:///qml/new-conversation.qml")
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

	window.On("sendMessage", func(to, subject, message string) {
		println("To: " + to)
		println("Subject: " + subject)
		println("Message: " + message)
	})

	window.Show()
	window.Wait()
	return nil
}
