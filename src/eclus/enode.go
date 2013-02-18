package main

import (
	"flag"
	"github.com/goerlang/node"
	"log"
)

var enableNode bool
var nodeName string
var nodeCookie string

func init() {
	flag.BoolVar(&enableNode, "node", false, "start erlang node")
	flag.StringVar(&nodeName, "node-name", "", "name of erlang node")
	flag.StringVar(&nodeCookie, "node-cookie", "", "cookie of erlang node")
}

func nodeEnabled() bool {
	return enableNode
}

func runNode() (enode *node.Node) {
	enode = node.NewNode(nodeName, nodeCookie)
	err := enode.Publish(5858)
	if err != nil {
		log.Printf("Cannot publish: %s", err)
		enode = nil
	}
	eSrv := new(eclusSrv)
	enode.Spawn(eSrv)
	return
}
