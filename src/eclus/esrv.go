package main

import (
	"github.com/goerlang/etf"
	"github.com/goerlang/node"
	"log"
)

type eclusSrv struct {
	node.GenServerImpl
}

func (es *eclusSrv) Init(args ...interface{}) {
	log.Printf("ECLUS_SRV: Init: %#v", args)
	es.Node.Register(etf.Atom("eclus"), es.Self)
}

func (es *eclusSrv) HandleCast(message *etf.Term) {
	log.Printf("ECLUS_SRV: HandleCast: %#v", *message)
}

func (es *eclusSrv) HandleCall(message *etf.Term, from *etf.Tuple) (reply *etf.Term) {
	log.Printf("ECLUS_SRV: HandleCall: %#v, From: %#v", *message, *from)
	replyTerm := etf.Term(etf.Tuple{etf.Atom("ok"), etf.Atom("eclus_reply"), *message})
	reply = &replyTerm
	return
}

func (es *eclusSrv) HandleInfo(message *etf.Term) {
	log.Printf("ECLUS_SRV: HandleInfo: %#v", *message)
}

func (es *eclusSrv) Terminate(reason interface{}) {
	log.Printf("ECLUS_SRV: Terminate: %#v", reason.(int))
}
