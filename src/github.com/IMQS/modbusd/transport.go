package modbusd

import (
	"bytes"
	"sync"
)

type Transport interface {
	Connect() error
	Send(*ADU) error
	Listen(chan bool) error
	Close() (err error)

	Lock()
	Unlock()
	Buffer() []byte
	Flush()
}

// Creates an instance of the TransportBase class
type TransportBase struct {
	Id       uint
	M        sync.Mutex
	Response bytes.Buffer
}

// Lock locks the mutex created for the transport to ensure mutually exclusive access to the response buffer
func (t *TransportBase) Lock() {
	t.M.Lock()
}

// Unlock unlocks the mutex created for the transport to ensure mutually exclusive access to the response buffer
func (t *TransportBase) Unlock() {
	t.M.Unlock()
}

// Buffer retrieves the content of the response buffer
func (t *TransportBase) Buffer() []byte {
	return t.Response.Bytes()
}

// Flush clears the content of the response buffer
func (t *TransportBase) Flush() {
	t.Response.Reset()
}
