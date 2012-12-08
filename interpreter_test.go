package gothic

import (
	"testing"
	"time"
)

var ir *Interpreter

func irinit(b *testing.B) {
	if ir == nil {
		ir = NewInterpreter(nil)
		time.Sleep(200 * time.Millisecond)
	}
	b.ResetTimer()
}

func BenchmarkTcl(b *testing.B) {
	irinit(b)

	ir.Set("N", b.N)
	ir.Eval(`
		for {set i 0} {$i < $N} {incr i} {
			set x 10
		}
	`)
}

func BenchmarkForeignGo(b *testing.B) {
	irinit(b)

	for i := 0; i < b.N; i++ {
		ir.Eval(`set x 10`)
	}
}

func BenchmarkNativeGo(b *testing.B) {
	irinit(b)

	ir.UnregisterCommand("test")
	ir.RegisterCommand("test", func() {
		for i := 0; i < b.N; i++ {
			ir.Eval(`set x 10`)
		}
	})
	ir.Eval(`test`)
}
