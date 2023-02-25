package main

import "io"

type Adapter struct {
	data    []byte
	readPos int
}

func NewAdapter() *Adapter {
	return &Adapter{
		data:    []byte{},
		readPos: 0,
	}
}

func (r *Adapter) Read(p []byte) (n int, err error) {
	if r.readPos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.readPos:])
	r.readPos += n
	return n, nil
}

func (r *Adapter) Fill(data []byte) {
	r.data = data
	r.readPos = 0
}
