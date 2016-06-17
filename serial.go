package modbusd

type Serial struct {
	TransportBase
	COM  string
	Baud uint32
}

// NewSerial creates an instance of the Serial class
func NewSerial(com string, baud uint32) (*Serial, error) {
	return &Serial{
		COM:  com,
		Baud: baud,
	}, nil
}

// Connect establishes a connection to a serial port device
func (s *Serial) Connect() error {
	return nil
}

// Send implements the transmission on the transport
func (s *Serial) Send() error {
	return nil
}

// Listen implements the recevier on the transport
func (s *Serial) Listen(done chan bool) error {
	for {
		select {
		case <-done:
			break
		}
	}

	return nil
}

// Close closes and cleans up after the connection
func (t *Serial) Close() error {
	return nil
}
