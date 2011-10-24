package main

import "fmt"
import "github.com/nsf/gothic"

func dispatcher(c <-chan string) {
	for v := range c {
		switch v {
		case "button1":
			fmt.Println("Button 1!")
		case "button2":
			fmt.Println("Button 2!")
		case "button3":
			fmt.Println("Button 3!")
		}
	}
}

func main() {
	ir, err := gothic.NewInterpreter()
	if err != nil {
		panic(err)
	}

	c := make(chan string)
	go dispatcher(c)

	ir.RegisterChannel("dispatcher", c)
	ir.Eval(`ttk::button .b1 -text "Button 1" -command {dispatcher <- button1}`)
	ir.Eval(`ttk::button .b2 -text "Button 2" -command {dispatcher <- button2}`)
	ir.Eval(`ttk::button .b3 -text "Button 3" -command {dispatcher <- button3}`)
	ir.Eval(`pack .b1 .b2 .b3`)
	ir.MainLoop()
}
