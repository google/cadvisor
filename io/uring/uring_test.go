package uring

import (
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

func TestReader(t *testing.T) {
	r := &reader{
		buffer: []byte("123"),
	}
	into := make([]byte, 2)
	count, err := r.Read(into)
	assert.Equal(t, 2, count)
	assert.NoError(t, err)

	into = make([]byte, 2)
	count, err = r.Read(into)
	assert.Equal(t, 1, count)
	assert.Equal(t, io.EOF, err)
}

func TestWriter(t *testing.T) {
	w := &writer{
		buffer:   make([]byte, 0, 3),
		capacity: 3,
	}

	count, err := w.Write([]byte("32"))
	assert.Equal(t, 2, count)
	assert.NoError(t, err)

	count, err = w.Write([]byte("10"))
	assert.Equal(t, 1, count)
	assert.Equal(t, io.EOF, err)
}
