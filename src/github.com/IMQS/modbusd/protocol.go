package modbusd

import (
	"encoding/binary"
	"fmt"
)

type Protocol interface {
	Encode(pdu *PDU) (*ADU, error)
	Decode([]byte) (*ADU, error)
}

type ProtocolBase struct {
	SlaveId byte
}

type State int
type Section int
type Element int

/*
 * Identified states, section states and element states employed to
 * effectively decode a modbus message in any of the supported
 * modbus protocol variations.
 */

const (
	TCOMPLETE State = 0
	TRESET    State = 1
	TPROCESS  State = 2
	TFAILURE  State = 3

	SMBAP  Section = 0
	SRTU   Section = 1
	SASCII Section = 2
	SPDU   Section = 3
	SERR   Section = 4
	SDONE  Section = 5
	SFAIL  Section = 6
	SSOF   Section = 7
	SEOF   Section = 8

	ENONE    Element = 0
	ETRANSID Element = 1
	EPROTOID Element = 2
	ESLAVEID Element = 3
	EFNCODE  Element = 4
	ELENGTH  Element = 5
	EEXCODE  Element = 6
	EDATA    Element = 7
	ECRC     Element = 8
	ELRC     Element = 9
)

// Recover implements a state machine that completely decodes all variations on the modbus protocol
func (p *ProtocolBase) Recover(response []byte, section Section) (*ADU, error) {
	var err error
	var adu *ADU
	/*
	 * The client's assigned protocol will assign the correct section the state machine has to
	 * be started in to enable the protocol base to decode the client's assigned protocol
	 * when fired from the Decode(Protocol interface) function.
	 */
	var start Section = section
	var state State = TRESET
	var element Element = ENONE
	var cnt byte
	var exitState, exitSection bool
	for {
		switch state {
		case TCOMPLETE:
			// All checks and balances on the decoded message has passed, return the ADU
			return adu, nil
		case TFAILURE:
			// A check or balance somewhere has failed
			return nil, fmt.Errorf("Failure to process response")
		case TRESET:
			// Starting condition of the state machine
			state = TPROCESS
		case TPROCESS:
			switch start {
			case SRTU:
				/*
				 * If the RTU protocol is employed start off with the error checking
				 * since a mismatch in CRC will immediately signal an invalid
				 * response and then there is no need to process elements.
				 */
				section = SERR
			case SMBAP:
				// Continue normally
			case SASCII:
				/*
				 * If the ASCII protocol is employed start off with the start of and end of
				 * frame sections to ensure raw data is framed correctly, then move to error checking
				 * since a non zero result in LRC will immediately signal an invalid
				 * response and then there is no need to process elements.
				 */
				section = SSOF
			default:
				return nil, fmt.Errorf("Unsupported protocol section code: %v", start)
			}
			// Decoding (processing) the entire raw message from the transport
			adu, err = NewADU(nil)
			if err != nil {
				return nil, fmt.Errorf("Unable to create ADU")
			}
			for {
				// Call on the relevant protocol header handler
				switch section {
				case SFAIL:
					state = TFAILURE
					exitSection = true
				case SDONE:
					state = TCOMPLETE
					exitSection = true
				case SSOF:
					if err = p.handleSOF(adu, response, start, &cnt, &section, &element); err != nil {
						return nil, err
					}
				case SEOF:
					if err = p.handleEOF(adu, response, start, &cnt, &section, &element); err != nil {
						return nil, err
					}
				case SMBAP:
					if err = p.handleMBAP(adu, response, &cnt, &section, &element); err != nil {
						return nil, err
					}
				case SRTU:
					if err = p.handleRTU(adu, response, &cnt, &section, &element); err != nil {
						return nil, err
					}
				case SASCII:
					if err = p.handleASCII(adu, response, &cnt, &section, &element); err != nil {
						return nil, err
					}
				case SPDU:
					if err = p.handlePDU(adu, response, &cnt, &start, &section, &element); err != nil {
						return nil, err
					}
				case SERR:
					// Error checking is handled for each type of modbus protocol
					switch start {
					case SMBAP:
						element = ENONE
						state = TCOMPLETE
						exitSection = true
					case SRTU:
						element = ECRC
					case SASCII:
						element = ELRC
					default:
						return nil, fmt.Errorf("Unsupported protocol section code: %v", start)
					}
					switch element {
					case ECRC:
						/*
						 * Only protocols that employ this error checking mechanism
						 * will call on this switch element.
						 */
						if err = p.handleCRC(adu, response, &cnt, &section, &element); err != nil {
							return nil, err
						}
					case ELRC:
						if err = p.handleLRC(adu, response, &cnt, &section, &element); err != nil {
							return nil, err
						}
					case ENONE:
						// This protocol does not employ an error checking mechanism
					default:
						return nil, fmt.Errorf("Unsupported element type (Error checking): %v", element)
					}
				default:
				}
				if exitSection {
					exitSection = false
					break
				}
			}

		default:
		}
		if exitState {
			exitState = false
			break
		}
	}

	return nil, fmt.Errorf("Unable to process response")
}

func (p *ProtocolBase) handleSOF(adu *ADU, response []byte, start Section, cnt *byte, section *Section, element *Element) error {
	switch start {
	case SMBAP:
	case SRTU:
	case SASCII:
		adu.SOF[0] = response[*cnt]
		*cnt++
		*section = SEOF
		if adu.SOF[0] != COLON {
			*section = SFAIL
			return fmt.Errorf("Frame alignment error (SOF) : %v", adu.SOF)
		}
	default:
		return fmt.Errorf("Protocol does not support a Start Of Frame : %v", start)
	}
	return nil
}

func (p *ProtocolBase) handleEOF(adu *ADU, response []byte, start Section, cnt *byte, section *Section, element *Element) error {
	switch start {
	case SMBAP:
	case SRTU:
	case SASCII:
		elen := int(LEOF)
		adu.EOF = response[len(response)-elen : len(response)]
		*section = SERR
		if adu.EOF[0] != CR || adu.EOF[1] != LF {
			*section = SFAIL
			return fmt.Errorf("Frame alignment error (EOF) : %v", adu.EOF)
		}
	default:
		return fmt.Errorf("Protocol does not support a End Of Frame : %v", start)
	}
	return nil
}

func (p *ProtocolBase) handleMBAP(adu *ADU, response []byte, cnt *byte, section *Section, element *Element) error {
	var exitElement = false
	*element = ETRANSID
	for {
		// Parse the various elements from the MBAP header
		switch *element {
		case ETRANSID:
			adu.TransactionId = binary.BigEndian.Uint16(response[*cnt:])
			binary.BigEndian.PutUint16(adu.Hdr[*cnt:], adu.TransactionId)
			*cnt = *cnt + byte(LTID)
			*element = EPROTOID
		case EPROTOID:
			adu.ProtocolId = binary.BigEndian.Uint16(response[*cnt:])
			binary.BigEndian.PutUint16(adu.Hdr[*cnt:], adu.ProtocolId)
			*cnt = *cnt + byte(LPID)
			*element = ELENGTH
		case ELENGTH:
			adu.Length = binary.BigEndian.Uint16(response[*cnt:])
			binary.BigEndian.PutUint16(adu.Hdr[*cnt:], adu.Length)
			*cnt = *cnt + byte(LLEN)
			*element = ESLAVEID
		case ESLAVEID:
			adu.SlaveId = response[*cnt]
			adu.Hdr[*cnt] = adu.SlaveId
			*cnt++
			// Skip out the the PDU processing section
			*section = SPDU
			exitElement = true
		default:
			return fmt.Errorf("Unsupported element type (MBAP): %v", element)
		}
		if exitElement {
			exitElement = false
			break
		}
	}
	return nil
}

func (p *ProtocolBase) handleRTU(adu *ADU, response []byte, cnt *byte, section *Section, element *Element) error {
	if len(response) < int(LSID+LFNC+LCRC) {
		return fmt.Errorf("Response header too short at %v bytes", len(response))
	}
	adu.Hdr = make([]byte, 1)
	*element = ESLAVEID
	// The RTU header contains only a SlaveId and is parsed from the repsonse
	switch *element {
	case ESLAVEID:
		adu.SlaveId = response[*cnt]
		adu.Hdr[*cnt] = adu.SlaveId
		*cnt++
		// Skip out the the PDU processing section
		*section = SPDU
	default:
		return fmt.Errorf("Unsupported element type (RTU): %v", *element)
	}
	return nil
}

func (p *ProtocolBase) handleASCII(adu *ADU, response []byte, cnt *byte, section *Section, element *Element) error {
	if len(response) < int(LSOF+LSID+LFNC+LLRC+LEOF) {
		return fmt.Errorf("Response header too short at %v bytes", len(response))
	}
	adu.Hdr = make([]byte, 1)
	*element = ESLAVEID
	// The RTU header contains only a SlaveId and is parsed from the repsonse
	switch *element {
	case ESLAVEID:
		adu.SlaveId = response[*cnt]
		adu.Hdr[*cnt] = adu.SlaveId
		*cnt++
		// Skip out the the PDU processing section
		*section = SPDU
	default:
		return fmt.Errorf("Unsupported element type (ASCII): %v", *element)
	}
	return nil
}

func (p *ProtocolBase) handleCRC(adu *ADU, response []byte, cnt *byte, section *Section, element *Element) error {
	/*
	 * Only protocols that employ this error checking mechanism
	 * will call on this switch element.
	 */
	var crc CRC
	clen := int(LCRC)
	if len(response) < (clen + int(LSID+LFNC)) {
		return fmt.Errorf("Response too short at %v bytes", len(response))
	}
	crcResponse := binary.BigEndian.Uint16(response[len(response)-clen:])
	adu.CRC = crcResponse
	adu.Err = make([]byte, clen)
	binary.BigEndian.PutUint16(adu.Err, crcResponse)
	crc.reset().calculate(response[0 : len(response)-clen])
	*section = SFAIL
	if crcResponse == crc.value() {
		*section = SRTU
	} else {
		return fmt.Errorf("Error checking CRC mismatch %v!=%v, %#v\n", adu.CRC, crcResponse, adu)
	}
	return nil
}

func (p *ProtocolBase) handleLRC(adu *ADU, response []byte, cnt *byte, section *Section, element *Element) error {
	*section = SASCII
	return nil
}

func (p *ProtocolBase) handlePDU(adu *ADU, response []byte, cnt *byte, start *Section, section *Section, element *Element) error {
	var exitElement = false
	*element = EFNCODE
	// Parse function code, exception code, length and or data from the PDU
	for {
		switch *element {
		case EFNCODE:
			fncode := FnCode(response[*cnt])
			adu.FnCode = []byte{byte(fncode)}
			*cnt++
			if FnCode(adu.FnCode[0]) >= FERR {
				*element = EEXCODE
				continue
			}
			switch FnCode(adu.FnCode[0]) {
			case RDCO, RDDI, RDIR, RDHR:
			default:
				return fmt.Errorf("Unsupported function code; %v", FnCode(adu.FnCode[0]))
			}

			// Set the next element expected in the response according to the type of protocol employed
			switch *start {
			case SMBAP:
				*element = EDATA
			case SRTU:
				*element = ELENGTH
			case SASCII:
			default:
				return fmt.Errorf("Unsupported protocol section code: %v", element)
			}
		case EEXCODE:
			/*
			 * An exception code has been identified from the function code value
			 * and has to be dealt with accordingly.
			 * Extract exception code from response.
			 */
			adu.ExceptionCode = ExCode(response[*cnt])
			adu.Data = make([]byte, 1) // just enough storage for excode
			adu.Data[0] = byte(adu.ExceptionCode)
			*cnt++

			// Take action on specific exception codes
			switch adu.ExceptionCode {
			case 0:
			case IllegalFunction:
			case IllegalDataAddress:
			case IllegalDataValue:
			case ServerDeviceFailure:
			case Acknowledge:
			case ServerDeviceBusy:
			case MemoryParityError:
			case GatewayPathUnavailable:
			case GatewayTargetDeviceFailedToRespond:
			default:
				return fmt.Errorf("Unsupported exception code: %v", adu.ExceptionCode)
			}
			// Retrieve exception code description
			if _, found := Exception[adu.ExceptionCode]; found {
				adu.Exception = Exception[adu.ExceptionCode]
			}
			*section = SDONE
			exitElement = true
		case ELENGTH:
			// Parse length from response
			adu.Length = uint16(response[*cnt])
			adu.Data = make([]byte, 1) // just enough storage for length
			adu.Data[0] = byte(adu.Length)
			*cnt++
			/*
			 * If a valid length is not encoded into the response message
			 * we immediately assume a response that has been received in error,
			 * although we could possibly calculate the data length from the
			 * response with a degree of uncertainty.
			 */
			if adu.Length <= 0 {
				return fmt.Errorf("Illegal length: %v", adu.Length)
			}
			*element = EDATA
		case EDATA:
			/*
			 * For the length retrieved in the ELENGTH element iterate
			 * and extract the raw data portion of the response.
			 */
			var idx byte
			for idx = 0; idx < byte(adu.Length); idx++ {
				adu.Data = append(adu.Data, response[*cnt])
				*cnt++
			}
			*section = SDONE
			exitElement = true
		default:
			return fmt.Errorf("Unsupported element type (PDU): %v", element)
		}
		if exitElement {
			exitElement = false
			break
		}
	}
	return nil
}
