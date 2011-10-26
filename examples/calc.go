package main

import "github.com/nsf/gothic"
import "big"

var args [2]*big.Int
var lastOp string
var afterOp = true

func applyOp(op string, text *gothic.StringVar) {
	num := text.Get()
	if args[0] == nil {
		if op != "=" {
			args[0] = big.NewInt(0)
			args[0].SetString(num, 10)
		}
	} else {
		args[1] = big.NewInt(0)
		args[1].SetString(num, 10)
	}

	afterOp = true

	if args[1] == nil {
		lastOp = op
		return
	}

	switch lastOp {
	case "+":
		args[0] = args[0].Add(args[0], args[1])
	case "-":
		args[0] = args[0].Sub(args[0], args[1])
	case "/":
		args[0] = args[0].Div(args[0], args[1])
	case "*":
		args[0] = args[0].Mul(args[0], args[1])
	}

	lastOp = op
	args[1] = nil

	text.Set(args[0].String())
	if op == "=" {
		args[0] = nil
	}
}

func main() {
	ir, err := gothic.NewInterpreter()
	if err != nil {
		panic(err)
	}

	lastOpVar := ir.NewStringVar("lastOp")
	calcTextVar := ir.NewStringVar("calcText")
	calcTextVar.Set("0")

	ir.RegisterCallback("appendNum", func(n string) {
		if afterOp {
			afterOp = false
			calcTextVar.Set("")
		}
		calcTextVar.Set(calcTextVar.Get() + n)
	})

	ir.RegisterCallback("applyOp", func(op string) {
		if afterOp && lastOp != "=" {
			return
		}
		applyOp(op, calcTextVar)
		lastOpVar.Set(lastOp)
	})

	ir.RegisterCallback("clearAll", func() {
		args[0] = nil
		args[1] = nil
		afterOp = true
		lastOp = ""
		lastOpVar.Set("")
		calcTextVar.Set("0")
	})

	ir.RegisterCallback("plusMinus", func() {
		text := calcTextVar.Get()
		if len(text) == 0 || text[0] == '0' {
			return
		}

		if text[0] == '-' {
			calcTextVar.Set(text[1:])
		} else {
			calcTextVar.Set("-" + text) 
		}
	})

	ir.Eval(`
wm title . "GoCalculator"
grid [ttk::frame .f] -column 0 -row 0 -columnspan 3 -sticky we
grid [ttk::entry .f.lastop -textvariable lastOp -justify center -state readonly -width 3] -column 0 -row 0 -sticky we
grid [ttk::entry .f.entry -textvariable calcText -justify right -state readonly] -column 1 -row 0 -sticky we
grid columnconfigure .f 0 -weight 0
grid columnconfigure .f 1 -weight 1
grid [ttk::button .0 -text 0 -command { appendNum 0 }] -column 0 -row 4 -sticky nwes
grid [ttk::button .1 -text 1 -command { appendNum 1 }] -column 0 -row 3 -sticky nwes
grid [ttk::button .2 -text 2 -command { appendNum 2 }] -column 1 -row 3 -sticky nwes
grid [ttk::button .3 -text 3 -command { appendNum 3 }] -column 2 -row 3 -sticky nwes
grid [ttk::button .4 -text 4 -command { appendNum 4 }] -column 0 -row 2 -sticky nwes
grid [ttk::button .5 -text 5 -command { appendNum 5 }] -column 1 -row 2 -sticky nwes
grid [ttk::button .6 -text 6 -command { appendNum 6 }] -column 2 -row 2 -sticky nwes
grid [ttk::button .7 -text 7 -command { appendNum 7 }] -column 0 -row 1 -sticky nwes
grid [ttk::button .8 -text 8 -command { appendNum 8 }] -column 1 -row 1 -sticky nwes
grid [ttk::button .9 -text 9 -command { appendNum 9 }] -column 2 -row 1 -sticky nwes
grid [ttk::button .pm    -text +/- -command plusMinus]   -column 1 -row 4 -sticky nwes
grid [ttk::button .clear -text C -command clearAll]      -column 2 -row 4 -sticky nwes
grid [ttk::button .eq    -text = -command { applyOp = }] -column 3 -row 4 -sticky nwes
grid [ttk::button .plus  -text + -command { applyOp + }] -column 3 -row 3 -sticky nwes
grid [ttk::button .minus -text - -command { applyOp - }] -column 3 -row 2 -sticky nwes
grid [ttk::button .mul   -text * -command { applyOp * }] -column 3 -row 1 -sticky nwes
grid [ttk::button .div   -text / -command { applyOp / }] -column 3 -row 0 -sticky nwes

foreach w [winfo children .] {grid configure $w -padx 3 -pady 3}

grid rowconfigure . 0 -weight 0
grid rowconfigure . 1 -weight 1
grid rowconfigure . 2 -weight 1
grid rowconfigure . 3 -weight 1
grid rowconfigure . 4 -weight 1
grid columnconfigure . 0 -weight 1
grid columnconfigure . 1 -weight 1
grid columnconfigure . 2 -weight 1
grid columnconfigure . 3 -weight 1

bind . 0             { appendNum 0 }
bind . <KP_Insert>   { appendNum 0 }
bind . 1             { appendNum 1 }
bind . <KP_End>      { appendNum 1 }
bind . 2             { appendNum 2 }
bind . <KP_Down>     { appendNum 2 }
bind . 3             { appendNum 3 }
bind . <KP_Next>     { appendNum 3 }
bind . 4             { appendNum 4 }
bind . <KP_Left>     { appendNum 4 }
bind . 5             { appendNum 5 }
bind . <KP_Begin>    { appendNum 5 }
bind . 6             { appendNum 6 }
bind . <KP_Right>    { appendNum 6 }
bind . 7             { appendNum 7 }
bind . <KP_Home>     { appendNum 7 }
bind . 8             { appendNum 8 }
bind . <KP_Up>       { appendNum 8 }
bind . 9             { appendNum 9 }
bind . <KP_Prior>    { appendNum 9 }
bind . +             { applyOp + }
bind . <KP_Add>      { applyOp + }
bind . -             { applyOp - }
bind . <KP_Subtract> { applyOp - }
bind . *             { applyOp * }
bind . <KP_Multiply> { applyOp * }
bind . /             { applyOp / }
bind . <KP_Divide>   { applyOp / }
bind . <Return>      { applyOp = }
bind . <KP_Enter>    { applyOp = }
bind . <BackSpace>   { clearAll }
	`)
	ir.MainLoop()
}
