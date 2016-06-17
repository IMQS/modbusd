package modbusd

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

const timeout time.Duration = 10 * time.Second

type TCP struct {
	TransportBase
	URL     string
	ip      string
	port    uint16
	Timeout time.Duration

	Conn net.Conn
}

// NewTCP creates an instance of the TCP transport class
func NewTCP(ip string, port uint16, to time.Duration) (*TCP, error) {
	return &TCP{
		URL:     fmt.Sprintf("%s:%s", ip, strconv.FormatUint(uint64(port), 10)),
		ip:      ip,
		port:    port,
		Timeout: to,
	}, nil
}

const RETRIES int = 5

// Connect opens a TCP transport connection
func (t *TCP) Connect() error {
	if t.Timeout <= 0 {
		t.Timeout = timeout
	}
	for retry := RETRIES; retry > 0; retry-- {
		dialer := net.Dialer{Timeout: t.Timeout}
		var err error
		if t.Conn, err = dialer.Dial("tcp", t.URL); err != nil {
			if retry == 1 {
				return fmt.Errorf("Could not dial URL %s: %s", t.URL, err)
			}
		} else {
			retry = 0
		}

	}
	return nil
}

// Send implements the transmission of transport
func (t *TCP) Send(adu *ADU) error {
	var err error
	if t.Conn == nil {
		// Establish a new connection and close it when complete
		if err = t.Connect(); err != nil {
			return err
		}
		defer t.Close()
	}

	if err = t.Conn.SetDeadline(time.Now().Add(t.Timeout)); err != nil {
		return err
	}

	var aduBytes []byte
	if aduBytes, err = adu.Bytes(); err != nil {
		return err
	}
	if _, err = t.Conn.Write(aduBytes); err != nil {
		return err
	}
	return nil
}

// Listen receives data on the transport
func (t *TCP) Listen(done chan bool) error {
	var err error
	var cnt int
	// Create response to accomodate maximum length response in a single read
	response := make([]byte, 256)
	cnt, err = t.Conn.Read(response)
	if err != nil {
		return err
	}
	if cnt > 0 {
		// Ensure exclusive access to the resource
		t.M.Lock()
		t.Response.Write(response[0:cnt])
		t.M.Unlock()
	}
	return nil
}

// Close closes and cleans up after a connection
func (t *TCP) Close() error {
	t.Conn.Close()
	return nil
}
