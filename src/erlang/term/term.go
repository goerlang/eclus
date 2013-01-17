package term

import (
	"log"
)

type ErlType byte

const (
	ErlTypeAtom         = ErlType('d')
	ErlTypeBinary       = 'm'
	ErlTypeCachedAtom   = 'C'
	ErlTypeFloat        = 'c'
	ErlTypeFun          = 'u'
	ErlTypeInteger      = 'b'
	ErlTypeLargeBig     = 'o'
	ErlTypeLargeTuple   = 'i'
	ErlTypeList         = 'l'
	ErlTypeNewCache     = 'N'
	ErlTypeNewFloat     = 'F'
	ErlTypeNewFun       = 'p'
	ErlTypeNewReference = 'r'
	ErlTypeNil          = 'j'
	ErlTypePid          = 'g'
	ErlTypePort         = 'f'
	ErlTypeReference    = 'e'
	ErlTypeSmallAtom    = 's'
	ErlTypeSmallBig     = 'n'
	ErlTypeSmallInteger = 'a'
	ErlTypeSmallTuple   = 'h'
	ErlTypeString       = 'k'
)

const (
	ErlFormatVersion = byte(131)
)

type Term interface{}
type Tuple []Term
type Array []Term

type Atom string

type Node Atom

type Pid struct {
	Node     Node
	Id       uint32
	Serial   uint32
	Creation byte
}

type Port struct {
	Node     Node
	Id       uint32
	Creation byte
}

type Reference struct {
	Node     Node
	Creation byte
	Id       []uint32
}

type Function struct {
	Arity     byte
	Unique    [16]byte
	Index     uint32
	Free      uint32
	Module    Atom
	OldIndex  uint32
	OldUnique uint32
	Pid       Pid
	FreeVars  []Term
}

type Export struct {
	Module   Atom
	Function Atom
	Arity    byte
}

func Read(buf []byte) (t *Term) {
	log.Printf("TODO: Read term...")
	return
}
