package main

import (
	"bytes"
	"encoding/binary"
	"erlang/dist"
	"flag"
	"log"
	"net"
	"strconv"
	"time"
)

var listenPort string
var regLimit int
var unregTTL int

func init() {
	flag.StringVar(&listenPort, "port", "4369", "listen port")
	flag.IntVar(&regLimit, "nodes-limit", 1000, "limit size of registration table to prune unregistered nodes")
	flag.IntVar(&unregTTL, "unreg-ttl", 10, "prune unregistered nodes if unregistration older than this value in minutes")
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
	*dist.NodeInfo
	Time   time.Time
	Active bool
	conn   net.Conn
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
						nReg[node].Active = false
						nReg[node].Time = now
					} else if rs > regLimit && !rec.Active && now.Sub(rec.Time).Minutes() > float64(unregTTL) {
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
				nInfo := dist.Read_ALIVE2_REQ(buf)

				var reply []byte

				if rec, ok := nReg[nInfo.Name]; ok {
					log.Printf("Node %s found", nInfo.Name)
					if rec.Active {
						log.Printf("Node %s is running", nInfo.Name)
						reply = dist.Compose_ALIVE2_RESP(false)
					} else {
						log.Printf("Node %s is not running", nInfo.Name)
						rec.conn = nConn

						nInfo.Creation = (rec.Creation % 3) + 1
						rec.NodeInfo = nInfo
						rec.Active = true
						reply = dist.Compose_ALIVE2_RESP(true, rec.Creation)
					}
				} else {
					log.Printf("New node %s", nInfo.Name)
					nInfo.Creation = 1
					rec := &nodeRec{
						NodeInfo: nInfo,
						conn:     nConn,
						Time:     time.Now(),
						Active:   true,
					}
					nReg[nInfo.Name] = rec
					reply = dist.Compose_ALIVE2_RESP(true, rec.Creation)
				}
				replyTo <- regAns{reply: reply, isClose: false}
			case dist.PORT_PLEASE2_REQ:
				nName := dist.Read_PORT_PLEASE2_REQ(buf)
				var reply []byte
				if rec, ok := nReg[nName]; ok && rec.Active {
					reply = dist.Compose_PORT2_RESP(rec.NodeInfo)
				} else {
					reply = dist.Compose_PORT2_RESP(nil)
				}
				replyTo <- regAns{reply: reply, isClose: true}
			case dist.NAMES_REQ, dist.DUMP_REQ:
				lp, err := strconv.Atoi(listenPort)
				if err != nil {
					log.Printf("Cannot convert %s to integer", listenPort)
					replyTo <- regAns{reply: nil, isClose: true}
				} else {
					replyB := new(bytes.Buffer)
					dist.Compose_START_NAMES_RESP(replyB, lp)
					for _, rec := range nReg {
						if dist.MessageId(buf[0]) == dist.NAMES_REQ {
							if rec.Active {
								dist.Append_NAMES_RESP(replyB, rec.NodeInfo)
							} else {
								if rec.Active {
									dist.Append_DUMP_RESP_ACTIVE(replyB, rec.NodeInfo)
								} else {
									dist.Append_DUMP_RESP_UNUSED(replyB, rec.NodeInfo)
								}
							}
						}
					}
					replyTo <- regAns{reply: replyB.Bytes(), isClose: true}
				}
			case dist.KILL_REQ:
				reply := dist.Compose_KILL_RESP()
				replyTo <- regAns{reply: reply, isClose: true}
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
