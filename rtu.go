package modbusd

const (
	LRTU Length = 1
)

type RTU struct {
	ProtocolBase
}

// Creates an instance of the RTU protocol class
func NewRTU(slaveid byte) (*RTU, error) {
	rtu := &RTU{}
	rtu.SlaveId = slaveid
	return rtu, nil
}

//Encode builds the RTU modbus protocol header and add error checking
func (r *RTU) Encode(pdu *PDU) (*ADU, error) {
	// Construct the Application Data Unit for the Modbus TCP protocol
	adu, err := NewADU(pdu)

	adu.Hdr = make([]byte, 1)
	// RTU header solely consists of the Slave Id
	adu.SlaveId = r.SlaveId
	adu.Hdr[0] = adu.SlaveId

	// Implement error checking for RTU
	adu.ErrorCRC()

	return adu, err
}

// decode calls on the ProtocolBase function
func (r *RTU) Decode(response []byte) (*ADU, error) {
	adu, err := r.Recover(response, SRTU)
	if err != nil {
		return nil, err
	}
	return adu, nil
}
