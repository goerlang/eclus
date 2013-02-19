package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"github.com/goerlang/epmd"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime/pprof"
	"strconv"
	"time"
)

var noEpmd bool
var listenPort string
var regLimit int
var unregTTL int
var cpuProfile string

func init() {
	flag.StringVar(&listenPort, "port", "4369", "listen port")
	flag.BoolVar(&noEpmd, "no-epmd", false, "disable epmd")
	flag.IntVar(&regLimit, "nodes-limit", 1000, "limit size of registration table to prune unregistered nodes")
	flag.IntVar(&unregTTL, "unreg-ttl", 10, "prune unregistered nodes if unregistration older than this value in minutes")
	flag.StringVar(&cpuProfile, "profile-cpu", "", "profile CPU to file")
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
	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	stopCh := make(chan bool)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			log.Printf("Signal %#v", sig)
			stopCh <- true
		}
	}()

	if !cliEnabled() {
		var err error
		var l net.Listener
		if !noEpmd {
			l, err = net.Listen("tcp", net.JoinHostPort("", listenPort))
		}
		if err != nil || !noEpmd {
			// Cannot bind, eclus instance already running, connect to it
			eclusCli()
		} else {
			if !noEpmd {
				epm := make(chan regReq, 10)
				go epmReg(epm)
				go func() {
					for {
						conn, err := l.Accept()
						log.Printf("Accept new")
						if err != nil {
							log.Printf(err.Error())
						} else {
							go mLoop(conn, epm)
						}
					}
				}()
			}
			if nodeEnabled() {
				go runNode()
			}
			<-stopCh
		}
	} else {
		eclusCli()
	}
}

type nodeRec struct {
	*epmd.NodeInfo
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
						nReg[node].conn = nil
					} else if rs > regLimit && !rec.Active && now.Sub(rec.Time).Minutes() > float64(unregTTL) {
						log.Printf("REG prune %s:%+v", node, rec)
						delete(nReg, node)
					}
				}
				continue
			}
			replyTo := req.replyTo
			log.Printf("IN: %v", buf)
			switch epmd.MessageId(buf[0]) {
			case epmd.ALIVE2_REQ:
				nConn := req.conn
				nInfo := epmd.Read_ALIVE2_REQ(buf)
				log.Printf("NodeInfo: %+v", nInfo)
				var reply []byte

				if rec, ok := nReg[nInfo.Name]; ok {
					log.Printf("Node %s found", nInfo.Name)
					if rec.Active {
						log.Printf("Node %s is running", nInfo.Name)
						reply = epmd.Compose_ALIVE2_RESP(false)
					} else {
						log.Printf("Node %s is not running", nInfo.Name)
						rec.conn = nConn

						nInfo.Creation = (rec.Creation % 3) + 1
						rec.NodeInfo = nInfo
						rec.Active = true
						reply = epmd.Compose_ALIVE2_RESP(true, rec.NodeInfo)
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
					reply = epmd.Compose_ALIVE2_RESP(true, rec.NodeInfo)
				}
				replyTo <- regAns{reply: reply, isClose: false}
			case epmd.PORT_PLEASE2_REQ:
				nName := epmd.Read_PORT_PLEASE2_REQ(buf)
				var reply []byte
				if rec, ok := nReg[nName]; ok && rec.Active {
					reply = epmd.Compose_PORT2_RESP(rec.NodeInfo)
				} else {
					reply = epmd.Compose_PORT2_RESP(nil)
				}
				replyTo <- regAns{reply: reply, isClose: true}
			case epmd.STOP_REQ:
				nName := epmd.Read_STOP_REQ(buf)
				var reply []byte
				if rec, ok := nReg[nName]; ok && rec.Active {
					// TODO: stop node
					reply = epmd.Compose_STOP_RESP(true)
				} else {
					reply = epmd.Compose_STOP_RESP(false)
				}
				replyTo <- regAns{reply: reply, isClose: true}
			case epmd.NAMES_REQ, epmd.DUMP_REQ:
				lp, err := strconv.Atoi(listenPort)
				if err != nil {
					log.Printf("Cannot convert %s to integer", listenPort)
					replyTo <- regAns{reply: nil, isClose: true}
				} else {
					replyB := new(bytes.Buffer)
					epmd.Compose_START_NAMES_RESP(replyB, lp)
					for _, rec := range nReg {
						if epmd.MessageId(buf[0]) == epmd.NAMES_REQ {
							if rec.Active {
								epmd.Append_NAMES_RESP(replyB, rec.NodeInfo)
							} else {
								if rec.Active {
									epmd.Append_DUMP_RESP_ACTIVE(replyB, rec.NodeInfo)
								} else {
									epmd.Append_DUMP_RESP_UNUSED(replyB, rec.NodeInfo)
								}
							}
						}
					}
					replyTo <- regAns{reply: replyB.Bytes(), isClose: true}
				}
			case epmd.KILL_REQ:
				reply := epmd.Compose_KILL_RESP()
				replyTo <- regAns{reply: reply, isClose: true}
			default:
				switch cliMessageId(buf[0]) {
				case REQ_NAMES:
					reply := ansNames(nReg)
					replyTo <- regAns{reply: reply, isClose: true}
				default:
					replyTo <- regAns{reply: nil, isClose: true}
				}
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
			log.Printf("Incomplete packet from erlang node to epmd: %d from %d", n, length)
			break
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
