package gothic

import (
	"bytes"
	"sync"
)

type buffer_pool_type struct {
	sync.Mutex
	buffers []bytes.Buffer
}

// always calls buffer.Reset() before returning it
func (bp *buffer_pool_type) get() bytes.Buffer {
	bp.Lock()
	if len(bp.buffers) == 0 {
		bp.Unlock()
		return bytes.Buffer{}
	}

	b := bp.buffers[len(bp.buffers)-1]
	bp.buffers = bp.buffers[:len(bp.buffers)-1]
	bp.Unlock()
	b.Reset()
	return b
}

func (bp *buffer_pool_type) put(b bytes.Buffer) {
	bp.Lock()
	bp.buffers = append(bp.buffers, b)
	bp.Unlock()
}

var buffer_pool buffer_pool_type
