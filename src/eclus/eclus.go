package main

import (
	"bytes"
	"encoding/binary"
	"erlang/dist"
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
)

var listenPort string

func init() {
	flag.StringVar(&listenPort, "port", "4369", "listen port")
}

type regAns struct {
	reply   []byte
	isClose bool
}

type regReq struct {
	buf     []byte
	replyTo chan regAns
	conn    net.Conn
}

func main() {
	flag.Parse()

	l, err := net.Listen("tcp", net.JoinHostPort("", listenPort))
	if err != nil {
		log.Fatal(err)
	}
	epm := make(chan regReq, 10)
	go epmReg(epm)
	for {
		conn, err := l.Accept()
		log.Printf("Accept new")
		if err != nil {
			log.Printf(err.Error())
		} else {
			go mLoop(conn, epm)
		}
	}
	log.Printf("Exit")
}

type nodeRec struct {
	dist.NodeInfo
	Time  time.Time
	Ready bool
	conn  net.Conn
}

func epmReg(in <-chan regReq) {

	var nReg = make(map[string]*nodeRec)

	for {
		select {
		case req := <-in:
			buf := req.buf
			if len(buf) == 0 {
				rs := len(nReg)
				log.Printf("REG %d records", rs)
				now := time.Now()

				for node, rec := range nReg {
					if rec.conn == req.conn {
						log.Printf("Connection for %s dropped", node)
						nReg[node].Ready = false
						nReg[node].Time = now
					} else if rs > 10 && !rec.Ready && now.Sub(rec.Time).Seconds() > 30 {
						log.Printf("REG prune %s:%+v", node, rec)
						delete(nReg, node)
					}
				}
				continue
			}
			replyTo := req.replyTo
			log.Printf("IN: %v", buf)
			switch dist.MessageId(buf[0]) {
			case dist.ALIVE2_REQ:
				nConn := req.conn
				nPort := binary.BigEndian.Uint16(buf[1:3])
				nType := buf[3]
				nProto := buf[4]
				highVsn := binary.BigEndian.Uint16(buf[5:7])
				lowVsn := binary.BigEndian.Uint16(buf[7:9])
				nLen := binary.BigEndian.Uint16(buf[9:11])
				offset := (11 + nLen)
				nName := string(buf[11:offset])
				//nELen := binary.BigEndian.Uint16(buf[offset:(offset+2)])
				nExtra := buf[(offset + 2):]
				log.Printf("Alive: N:%s, P:%d, T:%d", nName, nPort, nType)

				reply := make([]byte, 4)
				reply[0] = byte(dist.ALIVE2_RESP)
				reply[1] = 0 // OK

				var data uint16 = 0
				if rec, ok := nReg[nName]; ok {
					log.Printf("Node %s found", nName)
					if rec.Ready {
						log.Printf("Node %s is running", nName)
						reply[1] = 1 // ERROR
						data = 99    // CANNOT REGISTER
					} else {
						log.Printf("Node %s is not running", nName)
						rec.conn = nConn
						rec.Port = nPort
						rec.Type = nType
						rec.Protocol = nProto
						rec.HighVsn = highVsn
						rec.LowVsn = lowVsn
						rec.Extra = nExtra
						rec.Creation = (rec.Creation % 3) + 1
						rec.Ready = true
						data = rec.Creation
					}
				} else {
					log.Printf("New node %s", nName)
					rec := &nodeRec{
						NodeInfo: dist.NodeInfo{
							Name:     nName,
							Port:     nPort,
							Type:     nType,
							Protocol: nProto,
							HighVsn:  highVsn,
							LowVsn:   lowVsn,
							Extra:    nExtra,
							Creation: 1,
						},
						conn:  nConn,
						Time:  time.Now(),
						Ready: true,
					}
					nReg[nName] = rec
					data = rec.Creation
				}

				binary.BigEndian.PutUint16(reply[2:4], data)
				replyTo <- regAns{reply: reply, isClose: false}
			case dist.PORT_PLEASE2_REQ:
				nName := buf[1:]
				var reply []byte
				if rec, ok := nReg[string(nName)]; ok {
					reply = make([]byte, 14+len(nName)+len(rec.Extra))
					reply[0] = byte(dist.PORT2_RESP)
					reply[1] = 0 // OK
					binary.BigEndian.PutUint16(reply[2:4], rec.Port)
					reply[4] = rec.Type
					reply[5] = rec.Protocol
					binary.BigEndian.PutUint16(reply[6:8], rec.HighVsn)
					binary.BigEndian.PutUint16(reply[8:10], rec.LowVsn)
					nLen := len(rec.Name)
					binary.BigEndian.PutUint16(reply[10:12], uint16(nLen))
					offset := (12 + nLen)
					copy(reply[12:offset], rec.Name)
					nELen := len(rec.Extra)
					binary.BigEndian.PutUint16(reply[offset:offset+2], uint16(nELen))
					copy(reply[offset+2:offset+2+nELen], rec.Extra)
				} else {
					reply = make([]byte, 2)
					reply[0] = byte(dist.PORT2_RESP)
					reply[1] = 1 // ERROR
				}
				replyTo <- regAns{reply: reply, isClose: true}
			case dist.NAMES_REQ, dist.DUMP_REQ:
				log.Printf("NAMES_REQ")
				lp, err := strconv.Atoi(listenPort)
				if err != nil {
					log.Printf("Cannot convert %s to integer", listenPort)
					replyTo <- regAns{reply: nil, isClose: true}
				} else {
					log.Printf("Make a reply")

					var replyB bytes.Buffer
					reply := make([]byte, 4)
					binary.BigEndian.PutUint32(reply, uint32(lp))
					replyB.Write(reply)

					for node, rec := range nReg {
						if rec.Ready {
							if dist.MessageId(buf[0]) == dist.NAMES_REQ {
								replyB.Write([]byte(fmt.Sprintf("name %s at port %d\n", node, rec.Port)))
							} else {
								if rec.Ready {
									replyB.Write([]byte(fmt.Sprintf("active name     <%s> at port %d\n", node, rec.Port)))
								} else {
									replyB.Write([]byte(fmt.Sprintf("old/unused name <%s>, port = %d\n", node, rec.Port)))
								}
							}
						}
					}
					replyTo <- regAns{reply: replyB.Bytes(), isClose: true}
				}
			case dist.KILL_REQ:
				replyTo <- regAns{reply: []byte("OK"), isClose: true}
			default:
				replyTo <- regAns{reply: nil, isClose: true}
			}
		}
	}
}

func mLoop(c net.Conn, epm chan regReq) {
	buf := make([]byte, 1024)
	for {
		n, err := c.Read(buf)
		if err != nil {
			c.Close()
			log.Printf("Stop loop: %v", err)
			epm <- regReq{buf: []byte{}, conn: c}
			return
		}
		length := binary.BigEndian.Uint16(buf[0:2])
		if length != uint16(n-2) {
			log.Printf("Incomplete packet: %d from %d", n, length)
		}
		log.Printf("Read %d, %d: %v", n, length, buf[2:n])
		if isClose := handleMsg(c, buf[2:n], epm); isClose {
			break
		}
	}
	c.Close()
}

func handleMsg(c net.Conn, buf []byte, epm chan regReq) bool {
	myChan := make(chan regAns)
	epm <- regReq{buf: buf, replyTo: myChan, conn: c}
	select {
	case ans := <-myChan:
		log.Printf("Got reply: %+v", ans)
		if ans.reply != nil {
			c.Write(ans.reply)
		}
		return ans.isClose
	}
	return true
}
