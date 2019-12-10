package util

import (
	"fmt"
	"reflect"
	"unsafe"
)

// sliceBufferPool bytes buffer for reuse
type SliceBufferPool struct {
	queue chan []byte
	bufferSize int
	name string
}

func NewSliceBufferPool(name string, bufferSize int) (sbp *SliceBufferPool) {
	return &SliceBufferPool{
		queue: make(chan []byte, 256),
		bufferSize: bufferSize,
		name: name,
	}
}

func (sbp *SliceBufferPool) Enqueue(buffer []byte)  {
	// defer func() {
	// 	log.Debugf("after enqueue from %s, there is %d elements", sbp.name, len(sbp.queue))
	// }()

	if cap(buffer) < 1 {
		return
	}

	select {
	case sbp.queue <- buffer:
		return
	default:
		buffer = nil
	}
}

func (sbp *SliceBufferPool) DequeueWithInit(initSize int) (buffer []byte)  {
	if initSize >= sbp.bufferSize {
		panic(fmt.Sprintf("package size bigger than max buffer size need deal:%d",
			sbp.bufferSize))
	}

	defer func() {
		// reset cache byte
		pbytes := (*reflect.SliceHeader)(unsafe.Pointer(&buffer))
		pbytes.Len = initSize
	}()

	buffer = sbp.Dequeue()
	return
}

func (sbp *SliceBufferPool) Dequeue() (buffer []byte)  {
	// defer func() {
	// 	log.Debugf("after dequeue from %s, there is %d elements", sbp.name, len(sbp.queue))
	// }()

	select {
	case buffer = <- sbp.queue:
		return
	default:
		buffer = make([]byte, 0, sbp.bufferSize)
		return
	}
}
