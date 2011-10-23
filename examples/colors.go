package main

import "fmt"
import "github.com/nsf/gotk"

func main() {
	ir, err := gotk.NewInterpreter()
	if err != nil {
		panic(err)
	}

	entryText := ir.NewStringVar("entryText")
	ir.RegisterCallback("updateLabel", func() {
		ir.Eval(`.label configure -text {` + entryText.Get() + `}`)
		entryText.Set("")
	})

	var RGB [3]int
	ir.RegisterCallback("scaleUpdate", func(idx int, x float64) {
		if RGB[idx] == int(x) {
			return
		}
		RGB[idx] = int(x)
		col := fmt.Sprintf("%02X%02X%02X", RGB[0], RGB[1], RGB[2])
		ir.Eval(`.label configure -foreground #` + col)
	})

	ir.Eval(`ttk::button .hello -text "Press me!" -command updateLabel`)
	ir.Eval(`ttk::entry .entry -textvariable entryText`)
	ir.Eval(`ttk::label .label -text "Press a button"`)
	ir.Eval(`ttk::scale .scaleR -from 0 -to 255 -length 200 -command {scaleUpdate 0}`)
	ir.Eval(`ttk::scale .scaleG -from 0 -to 255 -length 200 -command {scaleUpdate 1}`)
	ir.Eval(`ttk::scale .scaleB -from 0 -to 255 -length 200 -command {scaleUpdate 2}`)
	ir.Eval(`pack .hello .entry .label .scaleR .scaleG .scaleB`)
	ir.MainLoop()
}
