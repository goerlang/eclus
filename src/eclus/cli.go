package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"flag"
	"log"
	"net"
	"sort"
	"time"
)

type cliMessageId uint8

const (
	REQ_NAMES  = cliMessageId('N') // 78
)

var isNames bool

func init() {
	flag.BoolVar(&isNames, "names", false, "(CLI) print names")
}


func epmCli() {
	c, err := net.Dial("tcp", net.JoinHostPort("", listenPort))
	if err != nil {
		log.Fatal("Cannot connect to %s port", listenPort)
	}
	var req []byte
	if isNames {
		req = reqNames()
	} else {
		c.Close()
		return
	}
	c.Write(req)
	buf := make([]byte, 1024)
	_, err = c.Read(buf)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf(string(buf))
	c.Close()
}


func reqNames() (req []byte) {
	req = make([]byte, 3)
	req[2] = byte('N')
	binary.BigEndian.PutUint16(req[0:2], 1)
	return
}

func ansNames(nReg map[string]*nodeRec) (reply []byte) {
	replyB := new(bytes.Buffer)
	mlen := 0
	var nodes = make(sort.StringSlice, len(nReg))
	var i int = 0
	for nn, _ := range nReg {
		nodes[i] = nn
		i++
		if len(nn) > mlen {
			mlen = len(nn)
		}
	}
	sort.Sort(&nodes)
	format := fmt.Sprintf("%%%ds\t%%d\t%%s\t%%d\t%%s\n", mlen)
	for _, nn := range nodes {
		rec := nReg[nn]
		replyB.Write([]byte(fmt.Sprintf(format, rec.Name, rec.Port, actStr(rec.Active), rec.Creation, rec.Time.Format(time.ANSIC))))
	}
	reply = replyB.Bytes()
	return
}

func actStr(a bool) (s string) {
	if a {
		s = "active"
	} else {
		s = "down"
	}
	return
}
