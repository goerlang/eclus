package dist

import (
	"bytes"
	"log"
	"encoding/binary"
	"crypto/md5"
	"io"
	"net"
	"errors"
	"strings"
	"strconv"
	"erlang/term"
)

type flagId uint32

const (
	PUBLISHED           = flagId(0x1)
	ATOM_CACHE          = 0x2
	EXTENDED_REFERENCES = 0x4
	DIST_MONITOR        = 0x8
	FUN_TAGS            = 0x10
	DIST_MONITOR_NAME   = 0x20
	HIDDEN_ATOM_CACHE   = 0x40
	NEW_FUN_TAGS        = 0x80
	EXTENDED_PIDS_PORTS = 0x100
	EXPORT_PTR_TAG      = 0x200
	BIT_BINARIES        = 0x400
	NEW_FLOATS          = 0x800
	UNICODE_IO          = 0x1000
	DIST_HDR_ATOM_CACHE = 0x2000
	SMALL_ATOM_TAGS     = 0x4000
)

type nodeFlag flagId

func (nf nodeFlag) toUint32() (flag uint32) {
	flag = uint32(nf)
	return
}

func (nf nodeFlag) isSet(f flagId) (is bool) {
	is = (uint32(nf) & uint32(f)) != 0
	return
}

func toNodeFlag(f ...flagId) (nf nodeFlag) {
	var flags uint32
	for _, v := range f {
		flags |= uint32(v)
	}
	nf = nodeFlag(flags)
	return
}

type nodeState uint8

const (
	HANDSHAKE    nodeState   = iota
	CONNECTED
)


type NodeDesc struct {
	Name   string
	Cookie string
	Hidden bool
	remote *NodeDesc
	state nodeState
	challenge uint32
	flag   nodeFlag
	version uint16
}

func NewNodeDesc(name, cookie string, isHidden bool) (nd *NodeDesc) {
	nd = &NodeDesc{
		Name: name,
		Cookie: cookie,
		Hidden: isHidden,
		remote: nil,
		state: HANDSHAKE,
		flag: toNodeFlag(PUBLISHED, UNICODE_IO, EXTENDED_PIDS_PORTS, EXTENDED_REFERENCES),
		version: 5,
	}
	return nd
}


func (currNd *NodeDesc) HandleMessage(c net.Conn) (err error) {
	buf := make([]byte, 1024)

	var n int
	n, err = c.Read(buf)
	if err != nil {
		c.Close()
		log.Printf("Stop enode loop (%d): %v", n, err)
		return
	}


	switch currNd.state {
	case HANDSHAKE:
		length := binary.BigEndian.Uint16(buf[0:2])
		msg := buf[2:n]
		log.Printf("Read from enode %d: %v", length, msg)

		sendData := func(data []byte) (int, error) {
			reply := make([]byte, len(data)+2)
			binary.BigEndian.PutUint16(reply[0:2], uint16(len(data)))
			copy(reply[2:], data)
			log.Printf("Write to enode: %v", reply)
			return c.Write(reply)
		}

		switch msg[0] {
		case 'n':
			sn := currNd.read_SEND_NAME(msg)
			// Statuses: ok, nok, ok_simultaneous, alive, not_allowed
			sok := currNd.compose_SEND_STATUS(sn, true)
			_, err = sendData(sok)
			if err != nil {
				return
			}

			currNd.challenge = 2112311231 // FIXME


			// Now send challenge
			challenge := currNd.compose_SEND_CHALLENGE(sn)
			sendData(challenge)
			if err != nil {
				return
			}

		case 'r':
			sn := currNd.remote
			ok := currNd.read_SEND_CHALLENGE_REPLY(sn, msg)
			if ok {
				challengeAck := currNd.compose_SEND_CHALLENGE_ACK(sn)
				sendData(challengeAck)
				if err != nil {
					return
				}
			} else {
				c.Close()
				err = errors.New("bad handshake")
				return
			}
		}
	case CONNECTED:
		length := binary.BigEndian.Uint32(buf[0:4])
		msg := buf[4:n]
		log.Printf("Read from enode %d: %v", length, msg)

		if length == 0 {
			log.Printf("Keepalive")
			return
		}

		switch msg[0] {
		case 'p':
			res := currNd.read_TERM(msg[1:])
				log.Printf("TERM: %v", res)
			}

	}
	return
}

func (nd *NodeDesc) compose_SEND_NAME() (msg []byte) {
	msg = make([]byte, 7+len(nd.Name))
	msg[0] = byte('n')
	binary.BigEndian.PutUint16(msg[1:3], nd.version)
	binary.BigEndian.PutUint32(msg[3:7], nd.flag.toUint32())
	copy(msg[7:], nd.Name)
	return
}

func (currNd *NodeDesc) read_SEND_NAME(msg []byte) (nd *NodeDesc) {
	version := binary.BigEndian.Uint16(msg[1:3])
	flag := nodeFlag(binary.BigEndian.Uint32(msg[3:7]))
	name := string(msg[7:])
	nd = &NodeDesc{
		Name: name,
		version: version,
		flag: flag,
	}
	currNd.remote = nd
	return
}

func (currNd *NodeDesc) compose_SEND_STATUS(nd *NodeDesc, isOk bool) (msg []byte) {
	msg = make([]byte, 3)
	msg[0] = byte('s')
	copy(msg[1:], "ok")
	return
}

func (currNd *NodeDesc) compose_SEND_CHALLENGE(nd *NodeDesc) (msg []byte) {
	msg = make([]byte, 11+len(currNd.Name))
	msg[0] = byte('n')
	binary.BigEndian.PutUint16(msg[1:3], currNd.version)
	binary.BigEndian.PutUint32(msg[3:7], currNd.flag.toUint32())
	binary.BigEndian.PutUint32(msg[7:11], currNd.challenge)
	copy(msg[11:], currNd.Name)
	return
}

func (currNd *NodeDesc) read_SEND_CHALLENGE_REPLY(nd *NodeDesc, msg []byte) (isOk bool) {
	nd.challenge = binary.BigEndian.Uint32(msg[1:5])
	digestB := msg[5:]

	digestA := genDigest(currNd.challenge, currNd.Cookie)
	if bytes.Compare(digestA, digestB) == 0 {
		isOk = true
		currNd.state = CONNECTED
	} else {
		isOk = false
	}
	return
}

func (currNd *NodeDesc) compose_SEND_CHALLENGE_ACK(nd *NodeDesc) (msg []byte) {
	msg = make([]byte, 17)
	msg[0] = byte('a')

	digestB := genDigest(nd.challenge, currNd.Cookie) // FIXME: use his cookie, not mine

	copy(msg[1:], digestB)
	return
}

func genDigest(challenge uint32, cookie string) (sum []byte) {
	h := md5.New()
	s := strings.Join([]string{cookie, strconv.FormatUint(uint64(challenge), 10)}, "")
	io.WriteString(h, s)
	sum = h.Sum(nil)
	return
}

func (nd NodeDesc) Flags() (flags []string) {
	fs := map[flagId]string{
		PUBLISHED           : "PUBLISHED",
		ATOM_CACHE          : "ATOM_CACHE",
		EXTENDED_REFERENCES : "EXTENDED_REFERENCES",
		DIST_MONITOR        : "DIST_MONITOR",
		FUN_TAGS            : "FUN_TAGS",
		DIST_MONITOR_NAME   : "DIST_MONITOR_NAME",
		HIDDEN_ATOM_CACHE   : "HIDDEN_ATOM_CACHE",
		NEW_FUN_TAGS        : "NEW_FUN_TAGS",
		EXTENDED_PIDS_PORTS : "EXTENDED_PIDS_PORTS",
		EXPORT_PTR_TAG      : "EXPORT_PTR_TAG",
		BIT_BINARIES        : "BIT_BINARIES",
		NEW_FLOATS          : "NEW_FLOATS",
		UNICODE_IO          : "UNICODE_IO",
		DIST_HDR_ATOM_CACHE : "DIST_HDR_ATOM_CACHE",
		SMALL_ATOM_TAGS     : "SMALL_ATOM_TAGS",
	}

	for k, v := range fs {
		if nd.flag.isSet(k) {
			flags = append(flags, v)
		}
	}
	return
}

func (currNd *NodeDesc) read_TERM(msg []byte) (t term.Term) {
	t = term.Read(msg)
	return
}
