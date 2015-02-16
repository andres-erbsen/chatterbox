//go:generate genqrc qml

package main

import (
	"fmt"
	"gopkg.in/qml.v1"
	"os"
)

func main() {
	if err := qml.Run(run); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

type Library struct {
	Count int
	Inventory []Book
}

type Book struct {
	Title string
	Author string
}

func run() error {

	engine := qml.NewEngine()

	controls, err := engine.LoadFile("qrc:///qml/new-conversation.qml")
	if err != nil {
		return err
	}

	mersault := Book{Title: "The Outsider", Author:"Camus"}
	lucky := Book{Title: "Waiting For Godot", Author:"Beckett"}
	kvothe := Book{Title: "The Name of the Wind", Author:"Rothfuss"}

	bookList := []Book{mersault, lucky, kvothe}

	library := Library{Count:3, Inventory:bookList}


	context := engine.Context()
	context.SetVar("library", &library)
	context.SetVar("book", &Book{Title: "The Outsider", Author:"Camus"})

	window := controls.CreateWindow(nil)
	listModel := window.ObjectByName("listModel")
	listModel.Call("addItem", "{\"title\": \"The Plague\", \"author\":\"Camus\"}")
	// window.On("sendMessage", func(to, subject, message string) {
	// 	println("To: " + to)
	// 	println("Subject: " + subject)
	// 	println("Message: " + message)
	// })

	window.Show()
	window.Wait()
	return nil
}
