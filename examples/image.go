package main

import "github.com/nsf/gothic"
import "image/png"
import "image"
import "os"

func loadPNG(filename string) image.Image {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	img, err := png.Decode(f)
	if err != nil {
		panic(err)
	}
	return img
}

func initGUI(ir *gothic.Interpreter) {
	ir.UploadImage("bg", loadPNG("background.png"))
	ir.Eval(`ttk::label .l -image bg`)
	ir.Eval(`pack .l -expand true`)
}

func main() {
	ir := gothic.NewInterpreter(initGUI)
	<-ir.Done
}
