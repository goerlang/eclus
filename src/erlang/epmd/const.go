package epmd

type MessageId uint8

const (
	ALIVE2_REQ  = MessageId('x') // 120
	ALIVE2_RESP = MessageId('y') // 121

	PORT_PLEASE2_REQ = MessageId('z') // 122
	PORT2_RESP       = MessageId('w') // 119

	NAMES_REQ = MessageId('n') // 110

	STOP_REQ = MessageId('s') // 115

	DUMP_REQ = MessageId('d') // 100

	KILL_REQ = MessageId('k') // 107

)
