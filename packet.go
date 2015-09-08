package agentx

import (
	"errors"
	"encoding/binary"
	"bytes"
	"log"
	"fmt"
	"io"
)

type AgentXPDUType uint8

const (
	agentx_Open		AgentXPDUType		= 1
	agentx_Close						= 2
	agentx_Register						= 3
    agentx_Unregister					= 4
    agentx_Get							= 5
    agentx_GetNext						= 6
    agentx_GetBulk						= 7
    agentx_TestSet						= 8
    agentx_CommitSet					= 9
    agentx_UndoSet						= 10
    agentx_CleanupSet					= 11
    agentx_Notify						= 12
    agentx_Ping							= 13
    agentx_IndexAllocate				= 14
    agentx_IndexDeallocate				= 15
    agentx_AddAgentCaps					= 16
    agentx_RemoveAgentCaps				= 17
    agentx_Response						= 18
)

func (t AgentXPDUType) String() string {
	switch {
	case t == agentx_Open: return "Open"
	case t == agentx_Close: return "Close"
	case t == agentx_Register: return "Register"
	case t == agentx_Unregister: return "Unregister"
	case t == agentx_Get: return "Get"
	case t == agentx_GetNext: return "GetNext"
	case t == agentx_GetBulk: return "GetBulk"
	case t == agentx_TestSet: return "TestSet"
	case t == agentx_CommitSet: return "CommitSet"
	case t == agentx_UndoSet: return "UndoSet"
	case t == agentx_CleanupSet: return "CleanupSet"
	case t == agentx_Notify: return "Notify"
	case t == agentx_Ping: return "Ping"
	case t == agentx_IndexAllocate: return "IndexAllocate"
	case t == agentx_IndexDeallocate: return "IndexDeallocate"
	case t == agentx_AddAgentCaps: return "AddAgentCaps"
	case t == agentx_RemoveAgentCaps: return "RemoveAgentCaps"
	case t == agentx_Response: return "Response"
	default: return "Invalid"
	}
}

type agentxCloseReason uint8

const (
	agentxCloseReason_Other				agentxCloseReason		= 1
	agentxCloseReason_ParseError								= 2
	agentxCloseReason_ProtocolError								= 3
	agentxCloseReason_Timeouts									= 4
	agentxCloseReason_Shutdown									= 5
	agentxCloseReason_ByManager									= 6
)

func (cr agentxCloseReason) String() string {
	switch {
	case cr == agentxCloseReason_Other: return "Other"
	case cr == agentxCloseReason_ParseError: return "Parse Error"
	case cr == agentxCloseReason_ProtocolError: return "Protocol Error"
	case cr == agentxCloseReason_Timeouts: return "Too many timeouts"
	case cr == agentxCloseReason_Shutdown: return "Shutting down"
	case cr == agentxCloseReason_ByManager: return "Shutdown by manager"
	default: return fmt.Sprintf("Unknown (%d)", cr)
	}
}

type Header struct {
	Version			uint8
	Type			AgentXPDUType
	Flags			uint8
	Reserved 		uint8
	SessionID		uint32
	TransactionID	uint32
	PacketID		uint32
	Length			uint32
}

type agentx_Open_PDU struct {
	hdr				Header
	Timeout			uint8

	n_subid			uint8
	prefix			uint8
	
	desc			string
	
	oid				[]OID
}

type agentx_Close_PDU struct {
	header			Header
	reason			agentxCloseReason
}

type agentx_Register_PDU struct {
	header			Header

	OID				OID
}

type agentx_Response_PDU struct {
	header			Header
	
	res_sysUpTime	uint32
	res_Error		agentXError
	res_Index		uint16
	
	VarBind			*VarBind
}

var (
	ErrBufferTooSmall		= errors.New("Buffer not large enough")
)

func unmarshalResponse(b []byte) (*agentx_Response_PDU, error) {
	res := &agentx_Response_PDU{}
	
	buf := bytes.NewBuffer(b)
	log.Printf("% x", b)
	
	err := binary.Read(buf, binary.LittleEndian, &res.res_sysUpTime)
	if err != nil {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &res.res_Error)
	if err != nil {
		return nil, err
	}

	err = binary.Read(buf, binary.LittleEndian, &res.res_Index)
	if err != nil {
		return nil, err
	}
	
	// unmarshal varbind list
	
	return res, nil
}

func (h *Header) Marshal(b *bytes.Buffer) error {

	err := binary.Write(b, binary.LittleEndian, h)
	if err != nil {
		log.Printf("Error marshaling data: %v", err)
		return err
	}

	return nil
}

func unmarshalHeader(b []byte) (*Header, error) {
	if len(b) < 20 {
		return nil, ErrBufferTooSmall
	}
	
	var h Header
	c := bytes.NewBuffer(b)
	err := binary.Read(c, binary.LittleEndian, &h)
	if err != nil {
		return nil, err
	}
	
	log.Printf("hdr: %v", h)
	
	return &h, nil
}

func (pdu *agentx_Open_PDU) Marshal() (*bytes.Buffer, error) {

	b := new(bytes.Buffer)

	err := binary.Write(b, binary.LittleEndian, pdu.Timeout)
	if err != nil {
		log.Printf("Error marshaling data: %v", err)
		return nil, err
	}

	// pad
	b.Write([]byte{0, 0, 0})
	
	err = binary.Write(b, binary.LittleEndian, pdu.n_subid)
	if err != nil {
		log.Printf("Error marshaling data: %v", err)
		return nil, err
	}

	err = binary.Write(b, binary.LittleEndian, pdu.prefix)
	if err != nil {
		log.Printf("Error marshaling data: %v", err)
		return nil, err
	}

	// pad
	b.Write([]byte{0, 0})

	l := uint32(len(pdu.desc))
	err = binary.Write(b, binary.LittleEndian, l)
	if err != nil {
		log.Printf("Error marshaling data: %v", err)
		return nil, err
	}

	err = binary.Write(b, binary.LittleEndian, []byte(pdu.desc))
	if err != nil {
		log.Printf("Error marshaling data: %v", err)
		return nil, err
	}

	return b, nil
}

func (pdu *agentx_Close_PDU) Marshal() (*bytes.Buffer, error) {
	b := new(bytes.Buffer)
	
	err := binary.Write(b, binary.LittleEndian, pdu.reason)
	if err != nil {
		log.Printf("Error marshaling data: %v", err)
		return nil, err
	}

	return b, nil
}

func (pdu *agentx_Register_PDU) Marshal() (*bytes.Buffer, error) {

	b := new(bytes.Buffer)

	l := uint8(len(pdu.OID))

	b.Write([]byte{0, 127, 0, 0, l, 0, 0, 0})
	
	for i := uint8(0); i < l; i++ {
		err := binary.Write(b, binary.LittleEndian, pdu.OID[i])
		if err != nil {
			log.Printf("Error marshaling data: %v", err)
			return nil, err
		}
		
	}
	
	return b, nil
}

type agentx_Get_PDU struct {
	OIDs			[]OID
}

var (
	retardedCompression = OID{1, 3, 6, 1, 4}
)

func unmarshalGet(b []byte) (*agentx_Get_PDU, error) {
	pdu := &agentx_Get_PDU{make([]OID, 0)}
	buf := bytes.NewBuffer(b)
	c := make([]uint8, 4)
	
	for {
		err := binary.Read(buf, binary.LittleEndian, c)
		if err != nil {
			log.Printf("Error: %v", err)
			if err == io.EOF {
				return pdu, nil
			}
			return nil, err
		}
		if (c[0] == 0) {
			return pdu, nil
		}
		log.Printf("HDR: %v", c)
		j := make(OID, c[0])
		err = binary.Read(buf, binary.LittleEndian, j)
		if err != nil {
			log.Printf("Error: %v", err)
			return nil, err
		}

		var o OID

		if c[1] > 0 {
			o = make(OID, 5 + c[0])
			// prepend 1.3.6.1.x
			o[0] = 1
			o[1] = 3
			o[2] = 6
			o[3] = 1
			o[4] = uint32(c[1])

			for i := uint8(0); i < c[0]; i++ {
				log.Printf("")
				o[i+5] = j[i]
			}
		} else {
			// o is j
			o = j
		}

		pdu.OIDs = append(pdu.OIDs, o)

		err = binary.Read(buf, binary.LittleEndian, c)
		if err != nil {
			log.Printf("Error: %v", err)
			return nil, err
		}
		
	}
}

func (pdu *agentx_Response_PDU) marshal(buf *bytes.Buffer) error {

	err := binary.Write(buf, binary.LittleEndian, &pdu.res_sysUpTime)
	if err != nil {
		return err
	}

	err = binary.Write(buf, binary.LittleEndian, &pdu.res_Error)
	if err != nil {
		return err
	}

	err = binary.Write(buf, binary.LittleEndian, &pdu.res_Index)
	if err != nil {
		return err
	}
	
	err = pdu.VarBind.marshal(buf)
	if err != nil {
		return err
	}
	
	return nil
}

func unmarshalGetNext(b []byte) (*agentx_Get_PDU, error) {
	pdu := &agentx_Get_PDU{make([]OID, 0)}
	buf := bytes.NewBuffer(b)
	c := make([]uint8, 4)
	var ignore = false
	
	for {
		err := binary.Read(buf, binary.LittleEndian, c)
		if err != nil {
			log.Printf("Error: %v", err)
			if err == io.EOF {
				return pdu, nil
			}
			return nil, err
		}
		if (c[0] == 0) {
			return pdu, nil
		}
		log.Printf("HDR: %v", c)
		j := make(OID, c[0])
		err = binary.Read(buf, binary.LittleEndian, j)
		if err != nil {
			log.Printf("Error: %v", err)
			return nil, err
		}

		var o OID

		if c[1] > 0 {
			o = make(OID, 5 + c[0])
			// prepend 1.3.6.1.x
			o[0] = 1
			o[1] = 3
			o[2] = 6
			o[3] = 1
			o[4] = uint32(c[1])

			for i := uint8(0); i < c[0]; i++ {
				log.Printf("")
				o[i+5] = j[i]
			}
		} else {
			// o is j
			o = j
		}

		if !ignore { pdu.OIDs = append(pdu.OIDs, o) }
		
		ignore = !ignore
	}
}
