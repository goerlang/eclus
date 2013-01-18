package epmd

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type NodeInfo struct {
	FullName string
	Name     string
	Domain   string
	Port     uint16
	Type     uint8 // 77 — Normal Erlang node, 72 — hidden (C-node)
	Protocol uint8
	HighVsn  uint16
	LowVsn   uint16
	Extra    []byte
	Creation uint16
}

func Read_ALIVE2_REQ(buf []byte) (ni *NodeInfo) {
	if len(buf) < 11 {
		return nil
	}
	nPort := binary.BigEndian.Uint16(buf[1:3])
	nType := buf[3]
	nProto := buf[4]
	highVsn := binary.BigEndian.Uint16(buf[5:7])
	lowVsn := binary.BigEndian.Uint16(buf[7:9])
	nLen := binary.BigEndian.Uint16(buf[9:11])

	offset := (11 + nLen)
	if len(buf) < int(offset+2) {
		return nil
	}
	nName := string(buf[11:offset])
	nELen := binary.BigEndian.Uint16(buf[offset:(offset + 2)])
	if len(buf) < int(offset+2+nELen) {
		return nil
	}
	nExtra := buf[(offset + 2):]

	return &NodeInfo{
		Name:     nName,
		Port:     nPort,
		Type:     nType,
		Protocol: nProto,
		HighVsn:  highVsn,
		LowVsn:   lowVsn,
		Extra:    nExtra,
		Creation: 0,
	}
}

func Compose_ALIVE2_REQ(ni *NodeInfo) (reply []byte) {
	reply = make([]byte, 14+len(ni.Name)+len(ni.Extra))
	reply[0] = byte(ALIVE2_REQ)
	binary.BigEndian.PutUint16(reply[1:3], ni.Port)
	reply[3] = ni.Type
	reply[4] = ni.Protocol
	binary.BigEndian.PutUint16(reply[5:7], ni.HighVsn)
	binary.BigEndian.PutUint16(reply[7:9], ni.LowVsn)
	nLen := len(ni.Name)
	binary.BigEndian.PutUint16(reply[9:11], uint16(nLen))
	offset := (11 + nLen)
	copy(reply[11:offset], ni.Name)
	nELen := len(ni.Extra)
	binary.BigEndian.PutUint16(reply[offset:offset+2], uint16(nELen))
	copy(reply[offset+2:offset+2+nELen], ni.Extra)
	return
}

func Compose_ALIVE2_RESP(isOk bool, value ...interface{}) (reply []byte) {
	reply = make([]byte, 4)
	reply[0] = byte(ALIVE2_RESP)
	if isOk {
		reply[1] = 0
		binary.BigEndian.PutUint16(reply[2:4], value[0].(*NodeInfo).Creation)
	} else {
		reply[1] = 1
		binary.BigEndian.PutUint16(reply[2:4], 99)
	}
	return
}

func Read_ALIVE2_RESP(reply []byte) (creation uint16, ok bool) {
	if reply[1] == 0 {
		ok = true
		creation = binary.BigEndian.Uint16(reply[2:4])
	} else {
		ok = false
		creation = binary.BigEndian.Uint16(reply[2:4])
	}
	return
}

func Read_PORT_PLEASE2_REQ(buf []byte) (name string) {
	name = string(buf[1:])
	return
}

func Compose_PORT2_RESP(ni *NodeInfo) (reply []byte) {
	if ni != nil {
		reply = make([]byte, 14+len(ni.Name)+len(ni.Extra))
		reply[0] = byte(PORT2_RESP)
		reply[1] = 0 // OK
		binary.BigEndian.PutUint16(reply[2:4], ni.Port)
		reply[4] = ni.Type
		reply[5] = ni.Protocol
		binary.BigEndian.PutUint16(reply[6:8], ni.HighVsn)
		binary.BigEndian.PutUint16(reply[8:10], ni.LowVsn)
		nLen := len(ni.Name)
		binary.BigEndian.PutUint16(reply[10:12], uint16(nLen))
		offset := (12 + nLen)
		copy(reply[12:offset], ni.Name)
		nELen := len(ni.Extra)
		binary.BigEndian.PutUint16(reply[offset:offset+2], uint16(nELen))
		copy(reply[offset+2:offset+2+nELen], ni.Extra)
	} else {
		reply = make([]byte, 2)
		reply[0] = byte(PORT2_RESP)
		reply[1] = 1 // ERROR
	}
	return
}

func Compose_START_NAMES_RESP(replyB *bytes.Buffer, port int) {
	reply := make([]byte, 4)
	binary.BigEndian.PutUint32(reply, uint32(port))
	replyB.Write(reply)
	return
}

func Append_NAMES_RESP(replyB *bytes.Buffer, ni *NodeInfo) {
	replyB.Write([]byte(fmt.Sprintf("name %s at port %d\n", ni.Name, ni.Port)))
}

func Append_DUMP_RESP_ACTIVE(replyB *bytes.Buffer, ni *NodeInfo) {
	replyB.Write([]byte(fmt.Sprintf("active name     <%s> at port %d\n", ni.Name, ni.Port)))
}

func Append_DUMP_RESP_UNUSED(replyB *bytes.Buffer, ni *NodeInfo) {
	replyB.Write([]byte(fmt.Sprintf("old/unused name <%s>, port = %d\n", ni.Name, ni.Port)))
}

func Compose_KILL_RESP() (reply []byte) {
	reply = []byte("OK")
	return
}

func Read_STOP_REQ(buf []byte) (name string) {
	name = string(buf[1:])
	return
}

func Compose_STOP_RESP(isOk bool) (reply []byte) {
	if isOk {
		reply = []byte("STOPPED")
	} else {
		reply = []byte("NOEXIST")
	}
	return
}
