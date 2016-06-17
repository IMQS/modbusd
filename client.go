package modbusd

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type ClientType string

// Known variations on the Modbus protocol that are supported
const (
	TCPCLIENT    ClientType = "TCPP" // TCP transport, MosbusTCP protocol
	ASCIIOVERTCP ClientType = "AOTP" // TCP transport, ASCII protocol (TODO: currently not supported)
	RTUOVERTCP   ClientType = "ROTP" // TCP transport, RTU protocol
	REMOTEUNIT   ClientType = "RTUP" // RTU transport, Serial protocol (TODO: currently not supported)
	TEXT         ClientType = "ASCP" // ASCII transport, Serial protocol (TODO: currently not supported)
)

type Client struct {
	Protocol  Protocol
	Transport Transport
}

// NewClient creates an instance of the Client class
func NewClient(u *URL) (*Client, error) {
	var err error
	var transport Transport
	var protocol Protocol
	/*
	 * The client will automatically assign the relevant protocol and transport according
	 * the definition of the ClientType.
	 */
	switch ClientType(strings.ToUpper(u.Protocol)) {
	case TCPCLIENT:
		// Assign a TCP transport and a ModbusTCP protocol
		if transport, err = NewTCP(u.IP, u.PortNo, time.Duration(u.Timeout)*time.Second); err != nil {
			return nil, fmt.Errorf("Unable to create transport: %s", err)
		}
		if protocol, err = NewModbusTCP(u.SlaveId); err != nil {
			return nil, fmt.Errorf("Unable to create protocol: %s", err)
		}
	case RTUOVERTCP:
		// Assign a TCP transport and a RTU protocol
		if transport, err = NewTCP(u.IP, u.PortNo, time.Duration(u.Timeout)*time.Second); err != nil {
			return nil, fmt.Errorf("Unable to create transport: %s", err)
		}
		if protocol, err = NewRTU(u.SlaveId); err != nil {
			return nil, fmt.Errorf("Unable to create protocol: %s", err)
		}
	case ASCIIOVERTCP:
		// TODO: Implement
	case REMOTEUNIT:
		// TODO: Implement
	case TEXT:
		// TODO: Implement

	default:
		// Impossible to establish client without transport and protocol definition
		return nil, fmt.Errorf("Unknown client type %s", ClientType(u.Protocol))
	}
	return &Client{
		Protocol:  protocol,
		Transport: transport,
	}, nil

}

func (c *Client) do(request *ADU) (*ADU, error) {
	var err error
	var response *ADU
	// Send a request encoded by client protocol
	if err = c.Transport.Send(request); err != nil {
		return nil, fmt.Errorf("Request failed: %s", err)
	}
	// Listen for a response in client protocol
	if err := c.Transport.Listen(nil); err != nil {
		if strings.Contains(err.Error(), "timeout") {
			var pdu *PDU
			pdu, err = NewPDU(FERR)
			if err != nil {
				return nil, fmt.Errorf("Transport unable to listen: %s", err)
			}
			response, err = NewADU(pdu)
			if err != nil {
				return nil, fmt.Errorf("Transport unable to listen: %s", err)
			}
			response.Timeout = true
			return response, nil
		} else {
			return nil, fmt.Errorf("Transport unable to listen: %s", err)
		}
	}

	/*
	 * Lock access to response buffer.
	 * Although not critical in command/response based communications,
	 * it is an important construct if any asynchronuous communications
	 * is employed.
	 */
	c.Transport.Lock()
	response, err = c.Protocol.Decode(c.Transport.Buffer())
	if err != nil {
		return nil, fmt.Errorf("Unable to decode response: %s", err)
	}
	// Clear buffer to avoid procesing the same data packet more than once
	c.Transport.Flush()
	c.Transport.Unlock()
	return response, nil
}

// Request connects to the modbus device, fires of a request and interprets the result
func (c *Client) Request(url *URL) (*ADU, error) {
	var err error
	if c == nil {
		return nil, fmt.Errorf("Illegal client")
	}

	if err := c.Transport.Connect(); err != nil {
		return nil, fmt.Errorf("Client connection failed: %s", err)
	}

	var response *ADU
	response, err = c.Read(url)
	if err != nil {
		return nil, fmt.Errorf("%s", err)
	}
	c.Close()
	return response, nil

}

func (c *Client) Read(url *URL) (*ADU, error) {
	var err error
	if c == nil {
		return nil, fmt.Errorf("Illegal client")
	}

	var request *Request
	var pdu *PDU
	var adu *ADU
	// Create a modbus request without any protocol encoding
	if request, err = NewRequest(FERR, strconv.FormatUint(url.Address, 10), url.Quantity); err != nil {
		return nil, fmt.Errorf("Unable to create modbus request: %s", err)
	}
	// Encode modbus request, still without protocol encoding
	if pdu, err = request.Encode(); err != nil {
		return nil, fmt.Errorf("Unable to encode modbus request: %#v. %s", request, err)
	}
	// Encode request in the assigned client protocol
	if adu, err = c.Protocol.Encode(pdu); err != nil {
		return nil, fmt.Errorf("Unable to encode the PDU: %#v, %s", pdu, err)
	}

	// Execute the request on a connection to the device identified by the transport
	var response *ADU
	response, err = c.do(adu)
	if err != nil {
		c.Transport.Lock()
		c.Transport.Flush()
		c.Transport.Unlock()
		return nil, fmt.Errorf("%s", err)
	}
	return response, nil

}

func (c *Client) Close() {
	// Close and dispose of connection
	c.Transport.Close()
}
