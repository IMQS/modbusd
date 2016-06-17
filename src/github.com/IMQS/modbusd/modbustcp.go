package modbusd

import (
	"encoding/binary"
	"fmt"
)

const (
	ModbusTCPProtocolId uint16 = 0x0000
)

const (
	LMBAP Length = 7
	LTID  Length = 2
	LPID  Length = 2
	LLEN  Length = 2
)

type ModbusTCP struct {
	MBAP
	ProtocolBase
}

type MBAP struct {
	TransactionId uint16
	ProtocolId    uint16
	Length        uint16
	//SlaveId is shared between protocols and thus included in ProtocolBase
}

// NewModbusTCP creates an instanc eof the ModbusTCP protocol class
func NewModbusTCP(slaveid byte) (*ModbusTCP, error) {
	m := &ModbusTCP{}
	// Can't assign to embedded struct literal so we access the member
	m.SlaveId = slaveid
	return m, nil
}

// Encode builds the MBAP header in the ModbusTCP protocol
func (m *ModbusTCP) Encode(pdu *PDU) (*ADU, error) {
	// Construct the Application Data Unit for the Modbus TCP protocol
	adu, err := NewADU(pdu)
	// Build the MBAP header
	// Transaction Id is simply a counter that increments for each request sent

	adu.TransactionId = m.TransactionId
	adu.Hdr = make([]byte, LMBAP)
	binary.BigEndian.PutUint16(adu.Hdr, adu.TransactionId)
	m.TransactionId++

	// Protocol Id is fixed from standard
	adu.ProtocolId = ModbusTCPProtocolId
	binary.BigEndian.PutUint16(adu.Hdr[LTID:], adu.ProtocolId)

	// Calculate length to include the Slave Id, Function Code and PDU Data payload
	adu.Length = uint16(LSID) + uint16(LFNC) + uint16(len(pdu.Data))
	if adu.Length <= 0 {
		return nil, fmt.Errorf("Invalid protocol length member: %v", adu.Length)
	}
	binary.BigEndian.PutUint16(adu.Hdr[(LTID+LPID):], adu.Length)

	// Device Slave Id
	adu.SlaveId = m.SlaveId
	adu.Hdr[(LTID+LPID)+LLEN] = adu.SlaveId

	// NOTE: No Error check implemented for Modbus TCP
	return adu, err
}

// Decode calls on the ProtocolBAse function Recover to decode received ModbusTCP protocol messages
func (m *ModbusTCP) Decode(response []byte) (*ADU, error) {
	adu, err := m.Recover(response, SMBAP)
	if err != nil {
		return nil, err
	}
	return adu, nil
}

