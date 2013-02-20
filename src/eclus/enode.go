package main

import (
	"flag"
	"github.com/goerlang/etf"
	"github.com/goerlang/node"
	"log"
)

var enableNode bool
var nodeName string
var nodeCookie string
var nodePort int

func init() {
	flag.BoolVar(&enableNode, "node", false, "start erlang node")
	flag.StringVar(&nodeName, "node-name", "", "name of erlang node")
	flag.StringVar(&nodeCookie, "node-cookie", "", "cookie of erlang node")
	flag.IntVar(&nodePort, "node-port", 5858, "port of erlang node")
}

func nodeEnabled() bool {
	return enableNode
}

func runNode() (enode *node.Node) {
	enode = node.NewNode(nodeName, nodeCookie)
	err := enode.Publish(nodePort)
	if err != nil {
		log.Printf("Cannot publish: %s", err)
		enode = nil
	}
	eSrv := new(eclusSrv)
	enode.Spawn(eSrv)

	eClos := func(terms etf.List) (r etf.Term) {
		r = etf.Term(etf.Tuple{etf.Atom("enode"), len(terms)})
		return
	}

	err = enode.RpcProvide("enode", "lambda", eClos)
	if err != nil {
		log.Printf("Cannot provide function to RPC: %s", err)
	}


	return
}
