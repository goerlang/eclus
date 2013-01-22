package main

import (
	erl "github.com/goerlang/etf/types"
	"github.com/goerlang/node"
	"log"
)

type eclusSrv struct {
	node.GenServerImpl
}

func (es *eclusSrv) Behaviour() (node.Behaviour, map[string]interface{}) {
	return es, es.Options()
}

func (es *eclusSrv) Init(args ...interface{}) {
	log.Printf("ECLUS_SRV: Init: %#v", args)
	es.Node.Register(erl.Atom("eclus"), es.Self)
}

func (es *eclusSrv) HandleCast(message *erl.Term) {
	log.Printf("ECLUS_SRV: HandleCast: %#v", *message)
}

func (es *eclusSrv) HandleCall(message *erl.Term, from *erl.Tuple) (reply *erl.Term) {
	log.Printf("ECLUS_SRV: HandleCall: %#v, From: %#v", *message, *from)
	replyTerm := erl.Term(erl.Tuple{erl.Atom("ok"), erl.Atom("eclus_reply")})
	reply = &replyTerm
	return
}

func (es *eclusSrv) HandleInfo(message *erl.Term) {
	log.Printf("ECLUS_SRV: HandleInfo: %#v", *message)
}

func (es *eclusSrv) Terminate(reason interface{}) {
	log.Printf("ECLUS_SRV: Terminate: %#v", reason.(int))
}
