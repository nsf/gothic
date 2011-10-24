package main

import "fmt"
import "github.com/nsf/gothic"

func main() {
	ir, err := gothic.NewInterpreter()
	if err != nil {
		panic(err)
	}

	var RGB [3]int
	ir.RegisterCallback("scaleUpdate", func(idx int, x float64) {
		if RGB[idx] == int(x) {
			return
		}
		RGB[idx] = int(x)
		col := fmt.Sprintf("%02X%02X%02X", RGB[0], RGB[1], RGB[2])
		ir.Eval(`ttk::style configure My.TFrame -background #` + col)
		ir.Eval(`.frame configure -style My.TFrame`)
	})

	ir.Eval(`
ttk::frame .frame -width 100 -height 30 -relief sunken

ttk::scale .scaleR -from 0 -to 255 -length 200 -command {scaleUpdate 0}
ttk::scale .scaleG -from 0 -to 255 -length 200 -command {scaleUpdate 1}
ttk::scale .scaleB -from 0 -to 255 -length 200 -command {scaleUpdate 2}
pack .frame -fill both
pack .scaleR .scaleG .scaleB
	`)
	ir.MainLoop()
}
