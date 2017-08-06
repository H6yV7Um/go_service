package utils

type LoopQueue struct {
	size 		int
	currentPos	int
	data 		[]interface{}
}

func (l *LoopQueue) New(size int) {
	if size <= 0 {
		return
	}
	l.size = size
	l.currentPos = 0
	l.data = make([]interface{}, size)
	// TODO: init l.data
}

func (l *LoopQueue) Push(v interface{}) {
	l.data[l.currentPos] = v
	if l.currentPos == l.size - 1 {
		l.currentPos = 0
	} else {
		l.currentPos++
	}
}

func (l *LoopQueue) Oldest() interface{} {
	return l.data[l.currentPos]
}

func (l *LoopQueue) Last() interface{} {
	return l.NBefore(1)
}

func (l *LoopQueue) NBefore(n int) interface{}{
	if n > l.size {
		n = l.size
	}
	if n > l.currentPos {
		return l.data[l.size - n + l.currentPos]
	} else {
		return l.data[l.currentPos - n]
	}
}
