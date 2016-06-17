package modbusd

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
)

type FnCode byte

const (
	/*
	 * Function code 0 is not mentioned in specification and is
	 * assumed an invalid code but used in this driver as the
	 * initialisation state of the Function Code
	 */
	INIT FnCode = 0x00 // INITialisation function code

	RDCO FnCode = 0x01 // ReaD COils
	RDDI FnCode = 0x02 // ReaD Discrete Inputs
	RDHR FnCode = 0x03 // ReaD Holding Registers
	RDIR FnCode = 0x04 // ReaD Input Registers
	WRSC FnCode = 0x05 // WRite Single Coil
	WRSR FnCode = 0x06 // WRite Single Register
	RDES FnCode = 0x07 // ReaD Exception Status (serial line only)
	DIAG FnCode = 0x08 // DIAGnostics (serial line only)

	GCEC FnCode = 0x0B // Get Comm Event Counter (serial line only)
	GCEL FnCode = 0x0C // Get Comm Event Log (serial line only)

	WRMC FnCode = 0x0F // WRite Multiple Coils
	WRMR FnCode = 0x10 // WRite Multiple Registers

	RSID FnCode = 0x11 // Report Slave ID (serial line only)

	RDFR FnCode = 0x14 // ReaD File Record
	WRFR FnCode = 0x15 // WRite File Record

	MWRR FnCode = 0x16 // Mask WRite Register
	RWMR FnCode = 0x17 // Read/Write Multiple Registers

	FIFO FnCode = 0x18 // read FIFO queue

	FERR FnCode = 0x80 // Add to FnCode to get erronous FnCode

	/* Currently unsupported function codes
		 * FnCode   NAME
		 * 0x14     Read File Record
		 * 0x15     Write File Record
	     * 0x17     Read/Write Multiple registers
	*/
)

type ExCode byte

// Known possible exception codes reported by devices
const (
	ExceptionZero                      ExCode = 0
	IllegalFunction                    ExCode = 1
	IllegalDataAddress                 ExCode = 2
	IllegalDataValue                   ExCode = 3
	ServerDeviceFailure                ExCode = 4
	Acknowledge                        ExCode = 5
	ServerDeviceBusy                   ExCode = 6
	MemoryParityError                  ExCode = 8
	GatewayPathUnavailable             ExCode = 10
	GatewayTargetDeviceFailedToRespond ExCode = 11
)

// Static description for each type of exception code that may be expected
var Exception map[ExCode]string = map[ExCode]string{
	IllegalFunction:                    "Illegal function code",
	IllegalDataAddress:                 "Illegal data address",
	IllegalDataValue:                   "Illegal data value",
	ServerDeviceFailure:                "Server device failure",
	Acknowledge:                        "Acknowledge",
	ServerDeviceBusy:                   "Server device busy",
	MemoryParityError:                  "Memory parity error",
	GatewayPathUnavailable:             "Gateway path unavailable",
	GatewayTargetDeviceFailedToRespond: "Gateway target device failed to respond",
}

type Length int

const (
	LSID Length = 1
	LFNC Length = 1
	LEXC Length = 1
	LCRC Length = 2
)

// Basic form of the modbus request, that aims to cover all possible modbus request forms
type Request struct {
	FnCode   FnCode
	Address  uint16 // Relative address
	Quantity uint16
	Count    byte
	Values   []uint16
	SubFn    uint16
	ANDMask  uint16
	ORMask   uint16
}

// NewRequest creates a new instance of a modbus Request class
func NewRequest(fncode FnCode, adr string, qty uint16) (*Request, error) {
	var err error
	var u64 uint64
	if u64, err = strconv.ParseUint(adr, 10, 64); err != nil {
		fmt.Errorf("Unable to parse address: %s", adr)
	}
	address, err := Relative(u64, &fncode)
	if err != nil {
		return nil, fmt.Errorf("Unable to map address to function code: %s", err)
	}
	return &Request{
		FnCode:   fncode,
		Address:  uint16(address),
		Quantity: qty,
	}, nil
}

// Function explicitly sets the function code of a modbus Request
func Relative(absolute uint64, fncode *FnCode) (uint64, error) {
	switch {
	case absolute >= 0 && absolute <= 65535:
		*fncode = RDCO
		return absolute, nil
	case absolute >= 100000 && absolute <= 165535:
		*fncode = RDDI
		return absolute - 100000, nil
	case absolute >= 300000 && absolute <= 365535:
		*fncode = RDIR
		return absolute - 300000, nil
	case absolute >= 400000 && absolute <= 465535:
		*fncode = RDHR
		return absolute - 400000, nil
	default:
		*fncode = FERR
		return absolute, fmt.Errorf("Unable to convert address to relative: %v", absolute)
	}
}

// Function explicitly sets the function code of a modbus Request
func Absolute(fncode FnCode, relative uint64) (uint64, error) {
	switch fncode {
	case RDCO:
		return relative, nil
	case RDDI:
		return relative + 100000, nil
	case RDIR:
		return relative + 300000, nil
	case RDHR:
		return relative + 400000, nil
	default:
		return relative, fmt.Errorf("Unable to convert address to absolute value: %v", relative)
	}
	return relative, nil
}

// Function explicitly sets the function code of a modbus Request
func (r *Request) Function(fncode FnCode) error {
	if fncode < FERR {
		r.FnCode = fncode
		return nil
	}
	return fmt.Errorf("Illegal function code: %v", fncode)
}

// Encode builds the modbus Request message
func (r *Request) Encode() (*PDU, error) {
	pdu, err := NewPDU(r.FnCode)
	switch r.FnCode {
	// Build the request for each function code individually or as groups for similar requests
	case RSID:
		// Requesting slave ID
		// A serial line only function code option
		pdu.Data = make([]byte, 0)
	case RDCO, RDDI, RDHR, RDIR:
		// Requesting coils, discrete inputs, input and holding registers
		pdu.Data = make([]byte, 4)
		binary.BigEndian.PutUint16(pdu.Data, r.Address)
		binary.BigEndian.PutUint16(pdu.Data[2:], r.Quantity)
	default:
		return &PDU{}, fmt.Errorf("Unsupported function code: %s", r.FnCode)
	}
	return pdu, err
}

// Application Data Unit
/*
 * The ADU structure should contain all possible values represented
 * by the available Modbus specification, with the relevant values
 * populated for each protocol
 */
type ADU struct {
	SOF []byte
	Hdr []byte

	MBAP
	SlaveId byte

	Err []byte
	CRC uint16

	ExceptionCode ExCode
	Exception     string

	Timeout bool

	PDU
	EOF []byte
}

// NewADU create an instance of the ADU class
func NewADU(pdu *PDU) (*ADU, error) {
	var err error
	if pdu == nil {
		/*
		 * Initialise PDU with erronuous function code.
		 * If the code is not corrected before protocol encoding,  this serves as
		 * a acheck to verify the encoding of the request.
		 */
		pdu, err = NewPDU(FERR)
		if err != nil {
			return nil, err
		}
	}
	return &ADU{
		PDU: *pdu,
	}, nil
}

// ErrorCheck calculates and assigns the error checking mechanism (CRC) to the ADU
func (d *ADU) ErrorCRC() error {
	var crc CRC
	var err error
	var aduBytes []byte
	/*
	 * Force to nil to ensure that aduBytes only contain the relevant
	 * information over which implement the error checking.
	 */
	d.Err = nil
	if aduBytes, err = d.Bytes(); err != nil {
		return fmt.Errorf("Unable to serialize ADU: %s", err)
	}
	/*
	 * Implement the error checking for an ADU.
	 * This function is only called if error checking is
	 * supported by the client's associated protocol.
	 */
	crc.reset().calculate(aduBytes)
	// Assign 16 bit value of CRC to the corresponding ADU member
	d.CRC = crc.value()
	d.Err = make([]byte, LCRC)
	// Assign CRC to error checking slice in ADU
	d.Err[0] = byte(d.CRC >> 8)
	d.Err[1] = byte(d.CRC)
	return nil
}

// ErrorCheck calculates and assigns the error checking mechanism (CRC) to the ADU
func (d *ADU) ErrorLRC() error {
	var err error
	var aduBytes []byte
	/*
	 * Force to nil to ensure that aduBytes only contain the relevant
	 * information over which implement the error checking.
	 */
	d.Err = nil
	if aduBytes, err = d.Bytes(); err != nil {
		return fmt.Errorf("Unable to serialize ADU: %s", err)
	}
	if aduBytes == nil {

	}
	/*
	 * Implement the error checking for an ADU.
	 * This function is only called if error checking is
	 * supported by the client's associated protocol.
	 */

	return nil
}

// Bytes converts the relevant members of an ADU to a single byte slice
func (d *ADU) Bytes() ([]byte, error) {
	/*
	 * Converts and appends relevant ADU fields to byte slice
	 * which in turn makes up the raw byte encoded message
	 * that is transmitted on the client transport.
	 */
	var err error
	w := new(bytes.Buffer)

	// Start of Frame
	if d.SOF != nil {
		if w.Write(d.SOF); err != nil {
			return nil, err
		}
	}
	// ADU header
	if d.Hdr != nil {
		if w.Write(d.Hdr); err != nil {
			return nil, err
		}
	}
	// Function code (part of the PDU)
	if d.FnCode != nil {
		if w.Write(d.FnCode); err != nil {
			return nil, err
		}
	}
	// Data (part of the PDU)
	if d.Data != nil {
		if w.Write(d.Data); err != nil {
			return nil, err
		}
	}
	// ADU error checking mechanism
	if d.Err != nil {
		if w.Write(d.Err); err != nil {
			return nil, err
		}
	}
	// End of Frame
	if d.EOF != nil {
		if w.Write(d.EOF); err != nil {
			return nil, err
		}
	}
	return w.Bytes(), nil
}

// Protocol Data Unit
type PDU struct {
	FnCode []byte
	Data   []byte
}

// NewPDU creates an instance of the PDU class
func NewPDU(fncode FnCode) (*PDU, error) {
	return &PDU{
		FnCode: []byte{byte(fncode)},
	}, nil
}
