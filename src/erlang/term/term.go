package term

import (
	"io"
	"log"
	"encoding/binary"
)

type ErlType byte

const (
	ErlTypeAtom         ErlType = 'd'
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

type Term interface{
	Write(io.Writer) error
}

type Tuple []Term

func (t Tuple) Write(w io.Writer) (err error) {
	if (len(t) >> 8) == 0 {
		w.Write([]byte{ErlTypeSmallTuple, byte(len(t))})
	} else {
		w.Write([]byte{ErlTypeLargeTuple, byte(len(t))})
	}
	for i := 0; i < len(t); i++ {
		err = t[i].Write(w)
		if err != nil {
			return
		}
	}
	return
}

type List []Term

func (t List) Write(w io.Writer) (err error) {
	for i := 0; i < len(t); i++ {
		err = t[i].Write(w)
		if err != nil {
			return
		}
	}
	return
}

type Int int32

func (t Int) Write(w io.Writer) (err error) {
	if (int32(t) >> 8) == 0 {
		_, err = w.Write([]byte{ErlTypeSmallInteger, uint8(t)})
	} else {
		buf := make([]byte, 5)
		buf[0] = ErlTypeSmallInteger
		binary.BigEndian.PutUint32(buf[1:], uint32(t))
		_, err = w.Write(buf)
	}
	return
}


type Atom string

func (t Atom) Write(w io.Writer) (err error) {
	buf := make([]byte, 3 + len(t))
	buf[0] = byte(ErlTypeAtom)
	binary.BigEndian.PutUint16(buf[1:3], uint16(len(t)))
	copy(buf[3:], []byte(t))
	_, err = w.Write(buf)
	return
}


type Pid struct {
	Node     Atom
	Id       uint32
	Serial   uint32
	Creation byte
}

func (t Pid) Write(w io.Writer) (err error) {
	// TODO: PID serialization
	return
}

type Port struct {
	Node     Atom
	Id       uint32
	Creation byte
}

type Reference struct {
	Node     Atom
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


func Read(buf []byte) (t Term) {
	if buf[0] != ErlFormatVersion {
		t = nil
		return
	}
	t, _ = readTerm(buf[1:])
	return
}

func readTerm(buf []byte) (t Term, n int) {
	switch ErlType(buf[0]) {
	case ErlTypeSmallTuple:
		t, n = readSmallTuple(buf)
	case ErlTypeSmallInteger:
		t, n = readSmallInteger(buf)
	case ErlTypePid:
		t, n = readPid(buf)
	case ErlTypeAtom:
		t, n = readAtom(buf)
	default:
		log.Printf("Term not supported: %v", buf)
	}
	return
}

func readSmallTuple(buf []byte) (t Tuple, n int) {
	arity := uint8(buf[1])
	log.Printf("Tuple arity: %d", arity)
	tuple := make([]Term, arity)
	n = 2
	var i uint8
	for i = 0; i < arity; i++ {
		var nr int
		tuple[i], nr = readTerm(buf[n:])
		n += nr
	}
	t = Tuple(tuple)
	return
}

func readSmallInteger(buf []byte) (t Int, n int) {
	t = Int(buf[1])
	n = 2
	return
}

func readPid(buf []byte) (t Pid, n int) {
	t.Node, n = readAtom(buf[1:])
	n += 1
	t.Id = binary.BigEndian.Uint32(buf[n:n+4])
	n += 4
	t.Serial = binary.BigEndian.Uint32(buf[n:n+4])
	n += 4
	t.Creation = buf[n]
	n += 1
	return
}

func readAtom(buf []byte) (t Atom, n int) {
	length := binary.BigEndian.Uint16(buf[1:3])
	t = Atom(buf[3:length+3])
	n = 3 + int(length)
	return
}
