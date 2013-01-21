package term

import (
	"encoding/binary"
	"flag"
	"io"
	"log"
)

var tTrace bool = false

func init() {
	flag.BoolVar(&tTrace, "erlang.term.trace", false, "trace erlang term format")
}

func tLog(f string, a ...interface{}) {
	if tTrace {
		log.Printf(f, a...)
	}
}

type ErlType byte

const (
	ErlTypeAtom         ErlType = 'd'
	ErlTypeBinary               = 'm'
	ErlTypeCachedAtom           = 'C'
	ErlTypeFloat                = 'c'
	ErlTypeFun                  = 'u'
	ErlTypeInteger              = 'b'
	ErlTypeLargeBig             = 'o'
	ErlTypeLargeTuple           = 'i'
	ErlTypeList                 = 'l'
	ErlTypeNewCache             = 'N'
	ErlTypeNewFloat             = 'F'
	ErlTypeNewFun               = 'p'
	ErlTypeNewReference         = 'r'
	ErlTypeNil                  = 'j'
	ErlTypePid                  = 'g'
	ErlTypePort                 = 'f'
	ErlTypeReference            = 'e'
	ErlTypeSmallAtom            = 's'
	ErlTypeSmallBig             = 'n'
	ErlTypeSmallInteger         = 'a'
	ErlTypeSmallTuple           = 'h'
	ErlTypeString               = 'k'
)

const (
	ErlFormatVersion = byte(131)
)

type Term interface {
	Write(io.Writer) error
}

func WriteTerm(t Term, w io.Writer) {
	w.Write([]byte{ErlFormatVersion})
	t.Write(w)
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

func (t Tuple) Element(n int) (ts Term) {
	// Start index from 1, like in Erlang
	ts = t[n-1]
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
	buf := make([]byte, 3+len(t))
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
	w.Write([]byte{ErlTypePid})
	t.Node.Write(w)
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, t.Id)
	w.Write(buf)
	binary.BigEndian.PutUint32(buf, t.Serial)
	w.Write(buf)
	w.Write([]byte{t.Creation})
	return
}

type Port struct {
	Node     Atom
	Id       uint32
	Creation byte
}

func (t Port) Write(w io.Writer) (err error) {
	// TODO: Port serialization
	return
}

type Reference struct {
	Node     Atom
	Creation byte
	Id       []uint32
}

func (t Reference) Write(w io.Writer) (err error) {
	w.Write([]byte{ErlTypeNewReference})
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(len(t.Id)))
	w.Write(buf)
	t.Node.Write(w)
	w.Write([]byte{t.Creation})

	buf = make([]byte, 4)
	for _, v := range t.Id {
		binary.BigEndian.PutUint32(buf, v)
		w.Write(buf)
	}
	return
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

func Read(buf []byte) (t Term, n int) {
	if (len(buf) == 0) || buf[0] != ErlFormatVersion {
		t = nil
		return
	}
	t, n = readTerm(buf[1:])
	n += 1
	return
}

func readTerm(buf []byte) (t Term, n int) {
	switch ErlType(buf[0]) {
	case ErlTypeSmallTuple:
		t, n = readSmallTuple(buf)
	case ErlTypeSmallInteger:
		t, n = readSmallInteger(buf)
	case ErlTypeInteger:
		t, n = readInteger(buf)
	case ErlTypePid:
		t, n = readPid(buf)
	case ErlTypeAtom:
		t, n = readAtom(buf)
	case ErlTypeNewReference:
		t, n = readNewReference(buf)
	case ErlTypeNil:
		t, n = readNil(buf)
	default:
		tLog("Term not supported: %v", buf)
	}
	return
}

func readSmallTuple(buf []byte) (t Tuple, n int) {
	arity := uint8(buf[1])
	tuple := make([]Term, arity)
	n = 2
	var i uint8
	for i = 0; i < arity; i++ {
		var nr int
		tuple[i], nr = readTerm(buf[n:])
		n += nr
	}
	t = Tuple(tuple)
	tLog("Read small tuple: %#v", t)
	return
}

func readSmallInteger(buf []byte) (t Int, n int) {
	t = Int(buf[1])
	n = 2
	tLog("Read small integer: %#v", t)
	return
}

func readInteger(buf []byte) (t Int, n int) {
	t = Int(binary.BigEndian.Uint32(buf[1:5]))
	n = 5
	tLog("Read integer: %#v", t)
	return
}

func readPid(buf []byte) (t Pid, n int) {
	t.Node, n = readAtom(buf[1:])
	n += 1
	t.Id = binary.BigEndian.Uint32(buf[n : n+4])
	n += 4
	t.Serial = binary.BigEndian.Uint32(buf[n : n+4])
	n += 4
	t.Creation = buf[n]
	n += 1
	tLog("Read pid: %#v", t)
	return
}

func readNewReference(buf []byte) (t Reference, n int) {
	length := binary.BigEndian.Uint16(buf[1:3])
	n += 3
	var pn int
	t.Node, pn = readAtom(buf[n:])
	n += pn

	t.Creation = buf[n]
	n += 1

	t.Id = make([]uint32, length)
	for i := 0; i < int(length); i++ {
		t.Id[i] = binary.BigEndian.Uint32(buf[n : n+4])
		n += 4
	}
	tLog("Read reference: %#v", t)
	return
}

func readAtom(buf []byte) (t Atom, n int) {
	length := binary.BigEndian.Uint16(buf[1:3])
	t = Atom(buf[3 : length+3])
	n = 3 + int(length)
	tLog("Read atom: %#v", t)
	return
}

func readNil(buf []byte) (t List, n int) {
	t = List(make([]Term, 0))
	n = 1
	tLog("Read empty list: %#v", t)
	return
}
