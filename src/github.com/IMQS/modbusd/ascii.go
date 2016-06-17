package modbusd

import (
	"encoding/hex"
	"fmt"
)

type ASCII struct {
	ProtocolBase
}

const (
	LSOF Length = 1
	LEOF Length = 2
	LLRC Length = 1
)

const COLON byte = 0x3A
const CR byte = 0x0D
const LF byte = 0x0A

//TODO: Implement the ASCII protocol
// Creates an instance of the RTU protocol class
func NewASCII(slaveid byte) (*ASCII, error) {
	ascii := &ASCII{}
	ascii.SlaveId = slaveid
	return ascii, nil
}

func (s *ASCII) toASCII(bin []byte) ([]byte, error) {
	str := fmt.Sprintf("%X", bin)
	var text []byte
	for idx := 0; idx < len(str); idx++ {
		text = append(text, byte(str[idx]))
	}
	return text, nil
}

func (s *ASCII) toHex(text []byte) ([]byte, error) {
	var str string
	for idx := 0; idx < len(text); idx++ {
		str += string(text[idx])
	}
	bhex, err := hex.DecodeString(str)
	if err != nil {
		return nil, err
	}
	return bhex, nil
}

//Encode builds the ASCII modbus protocol header and add error checking
func (s *ASCII) Encode(pdu *PDU) (*ADU, error) {
	// Construct the Application Data Unit for the Modbus TCP protocol
	adu, err := NewADU(pdu)

	// Start of frame control characters
	adu.SOF = make([]byte, 1)
	adu.SOF[0] = COLON
	if adu.SOF, err = s.toASCII(adu.SOF); err != nil {
		return nil, fmt.Errorf("Unable to encode header")
	}

	// RTU header solely consists of the Slave Id
	adu.Hdr = make([]byte, 1)
	adu.SlaveId = s.SlaveId
	adu.Hdr[0] = adu.SlaveId

	/*
	 * Convert all of the ADU sections to their ASCII
	 * representation that the ASCII modbus protocol
	 * requires.
	 */
	if adu.Hdr, err = s.toASCII(adu.Hdr); err != nil {
		return nil, fmt.Errorf("Unable to encode header")
	}

	if adu.FnCode, err = s.toASCII(adu.FnCode); err != nil {
		return nil, fmt.Errorf("Unable to encode function code")
	}

	if adu.Data, err = s.toASCII(adu.Data); err != nil {
		return nil, fmt.Errorf("Unable to encode data")
	}

	// Implement error checking for ASCII
	adu.ErrorLRC()
	if adu.Err, err = s.toASCII(adu.Err); err != nil {
		return nil, fmt.Errorf("Unable to encode error checking")
	}

	// End of frame control characters
	adu.EOF = make([]byte, 2)
	adu.EOF[0] = CR
	adu.EOF[1] = LF
	if adu.EOF, err = s.toASCII(adu.EOF); err != nil {
		return nil, fmt.Errorf("Unable to encode header")
	}

	return adu, err
}

// decode calls on the ProtocolBase function
func (s *ASCII) Decode(response []byte) (*ADU, error) {
	var err error
	var tmp *ADU
	/*
	 * Convert all of the ADU sections to their hexadecimal (binary)
	 * representation.
	 */
	if tmp.SOF, err = s.toHex(response[:2]); err != nil {
		return nil, fmt.Errorf("Unable to encode header")
	}

	if tmp.Hdr, err = s.toHex(response[2:4]); err != nil {
		return nil, fmt.Errorf("Unable to encode header")
	}

	if tmp.FnCode, err = s.toHex(response[4:6]); err != nil {
		return nil, fmt.Errorf("Unable to encode function code")
	}

	if tmp.Data, err = s.toHex(response[6 : len(response)-6]); err != nil {
		return nil, fmt.Errorf("Unable to encode data")
	}

	if tmp.Err, err = s.toHex(response[len(response)-6 : len(response)-4]); err != nil {
		return nil, fmt.Errorf("Unable to encode error checking")
	}

	if tmp.EOF, err = s.toHex(response[len(response)-4 : len(response)]); err != nil {
		return nil, fmt.Errorf("Unable to encode header")
	}

	var tmpBytes []byte
	tmpBytes, err = tmp.Bytes()
	if err != nil {
		return nil, err
	}

	/*
	 * The recover function should be able to handle the
	 * message structure if the bytes have been converted from the ASCII
	 * represetation to the binary.
	 */
	adu, err := s.Recover(tmpBytes, SASCII)
	if err != nil {
		return nil, err
	}
	return adu, nil
}
