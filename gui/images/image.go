package images

import (
	"bytes"
	"image"
	"image/draw"
	_ "image/png"
)

func Get(name string) *image.RGBA {
	data, err := Asset(name)
	if err != nil {
		panic(err)
	}
	png, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		panic("error loading image")
	}
	img := image.NewRGBA(png.Bounds())
	draw.Draw(img, img.Rect, png, image.ZP, draw.Src)
	return img
}
