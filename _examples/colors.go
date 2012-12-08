package main

import "github.com/nsf/gothic"

func main() {
	ir := gothic.NewInterpreter(`
		ttk::style configure My.TFrame -background #000000
		ttk::frame .frame -width 100 -height 30 -relief sunken -style My.TFrame

		ttk::scale .scaleR -from 0 -to 255 -length 200 -command {scaleUpdate 0}
		ttk::scale .scaleG -from 0 -to 255 -length 200 -command {scaleUpdate 1}
		ttk::scale .scaleB -from 0 -to 255 -length 200 -command {scaleUpdate 2}

		pack .frame -fill both -expand true
		pack .scaleR .scaleG .scaleB -fill both
	`)

	var RGB [3]int
	ir.RegisterCommand("scaleUpdate", func(idx int, x float64) {
		if RGB[idx] == int(x) {
			return
		}
		RGB[idx] = int(x)
		ir.Eval(`ttk::style configure My.TFrame -background `+
			`#%{%02X}%{%02X}%{%02X}`, RGB[0], RGB[1], RGB[2])
	})

	<-ir.Done
}
