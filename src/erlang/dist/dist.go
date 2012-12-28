package dist

import (
	// "os"
	// "net"
	// "log"
	//"time"
)

type NodeInfo struct {
	Name       string
	Port       uint16
	Type       uint8
	Protocol   uint8
	HighVsn    uint16
	LowVsn     uint16
	Extra      []byte
	Creation   uint16
}
