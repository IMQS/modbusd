package modbusd

import (
	"fmt"
	"strconv"
	"strings"
)

type URL struct {
	SURL     string
	IP       string
	PortNo   uint16
	SlaveId  byte
	Timeout  uint
	Protocol string
	Address  uint64 // Absolute address
	Quantity uint16
}

func NewURL(surl string) (*URL, error) {
	url := &URL{}
	url.SURL = surl
	components := strings.Split(surl, "//")
	if len(components) != 2 {
		return nil, fmt.Errorf("Invalid URL: %s", surl)
	}
	url.Protocol = strings.Split(components[0], ":")[0]
	components = strings.Split(components[1], "/")
	if len(components) != 3 {
		return nil, fmt.Errorf("Invalid URL: %s", surl)
	}
	url.IP = strings.Split(components[0], ":")[0]

	var err error
	var u64 uint64
	if u64, err = strconv.ParseUint(strings.Split(components[0], ":")[1], 10, 16); err != nil {
		return nil, fmt.Errorf("Unable to parse Port number: %s", err)
	}
	url.PortNo = uint16(u64)
	if u64, err = strconv.ParseUint(strings.Split(components[1], "-")[0], 10, 8); err != nil {
		return nil, fmt.Errorf("Unable to parse Slave Id: %s", err)
	}
	url.SlaveId = byte(u64)
	if u64, err = strconv.ParseUint(strings.Split(components[1], "-")[1], 10, 32); err != nil {
		return nil, fmt.Errorf("Unable to parse Timeout: %s", err)
	}
	url.Timeout = uint(u64)
	if u64, err = strconv.ParseUint(strings.Split(components[2], "-")[0], 10, 64); err != nil {
		return nil, fmt.Errorf("Unable to parse Address: %s", err)
	}
	url.Address = u64
	if u64, err = strconv.ParseUint(strings.Split(components[2], "-")[1], 10, 16); err != nil {
		return nil, fmt.Errorf("Unable to parse Quantity: %s", err)
	}
	url.Quantity = uint16(u64)

	return url, nil
}
