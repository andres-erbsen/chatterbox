package main

import (
	"github.com/andlabs/ui"
	"github.com/andres-erbsen/chatterbox/gui/images"
	"image"
	"log"
)

func main() {
	go ui.Do(guiInit)
	if err := ui.Go(); err != nil {
		log.Fatal(err)
	}
}

func centered(c ui.Control) ui.Control {
	ret := ui.NewGrid()
	ret.Add(c, nil, 0, true, ui.Center, true, ui.Center, 1, 1)
	return ret
}

type imageHandler image.RGBA

func (img *imageHandler) Paint(rect image.Rectangle) *image.RGBA {
	return (*image.RGBA)(img).SubImage(rect).(*image.RGBA)
}
func (img *imageHandler) Mouse(me ui.MouseEvent)  {}
func (img *imageHandler) Key(ke ui.KeyEvent) bool { return false }

func guiInit() {
	history := ui.NewTextbox()
	history.SetText("history")

	input := ui.NewTextField()
	// unfortunately we cannot detent Enter presses... <https://github.com/andlabs/ui/issues/43>
	// input.OnChanged(func() {
	//	println("changed")
	// })
	send := ui.NewButton("Send")
	send.OnClicked(func() {
		history.SetText(history.Text() + "\n" + input.Text())
		input.SetText("")
	})

	status := ui.NewArea(16, 16, (*imageHandler)(images.Get("blurry.png")))

	bottomRow := ui.NewHorizontalStack(input, centered(status), send)
	bottomRow.SetStretchy(0)
	st := ui.NewVerticalStack(history, bottomRow)
	st.SetStretchy(0)

	w := ui.NewWindow("Chatterbox conversation", 160, 800, st)

	w.OnClosing(func() bool {
		ui.Stop()
		return true
	})
	w.Show()
}
