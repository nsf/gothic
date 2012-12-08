package gothic

import (
	"testing"
	"time"
)

var ir *Interpreter

func init() {
	ir = NewInterpreter(nil)
	time.Sleep(200 * time.Millisecond)
}

func BenchmarkTcl(b *testing.B) {
	ir.Set("N", b.N)
	ir.Eval(`
		for {set i 1} {$i < $N} {incr i} {
			set x 10
		}
	`)
}

func BenchmarkForeignGo(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ir.Eval(`set x 10`)
	}
}

func BenchmarkNativeGo(b *testing.B) {
	ir.UnregisterCommand("test")
	ir.RegisterCommand("test", func() {
		for i := 0; i < b.N; i++ {
			ir.Eval(`set x 10`)
		}
	})
	ir.Eval(`test`)
}
