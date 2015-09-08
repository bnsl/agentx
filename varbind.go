package agentx

import (
	"sync"
	"bytes"
	"encoding/binary"
)

type varBindType uint16

const (
	vbInteger			varBindType			= 2
	vbOctetString							= 4
	vbNull                     				= 5
	vbObjectIdentifier        				= 6
	vbIpAddress               				= 64
	vbCounter32               				= 65
	vbGauge32                 				= 66
	vbTimeTicks               				= 67
	vbOpaque                  				= 68
	vbCounter64               				= 70
	vbnoSuchObject           				= 128
	vbnoSuchInstance         				= 129
	vbendOfMibView           				= 130
)

type VarBindEntry struct {
	Type varBindType
	OID OID
	Data SNMPValue
}

type VarBind struct {
	entries []VarBindEntry
	lock sync.RWMutex
}

func (vbt varBindType) String() string {
	switch {
	case vbt == vbInteger: return "Integer"
	case vbt == vbOctetString: return "Octet String"
	case vbt == vbNull: return "Null"
	case vbt == vbObjectIdentifier: return "Object Identifier"
	case vbt == vbIpAddress: return "IP Address"
	case vbt == vbCounter32: return "Counter 32"
	case vbt == vbGauge32: return "Gauge 32"
	case vbt == vbTimeTicks: return "TimeTicks"
	case vbt == vbOpaque: return "Opaque"
	case vbt == vbCounter64: return "Counter 64"
	case vbt == vbnoSuchObject: return "noSuchObject"
	case vbt == vbnoSuchInstance: return "noSuchInstance"
	case vbt == vbendOfMibView: return "endOfMibView"
	default: return "UNKNOWN"
	}
}

func NewVarBind() *VarBind {
	return &VarBind{entries: make([]VarBindEntry, 0)}
}

func (vb *VarBind) Add(typ varBindType, oid OID, data SNMPValue) {
	vb.lock.Lock()
	defer vb.lock.Unlock()

	vb.entries = append(vb.entries, VarBindEntry{typ, oid, data})
}

func (vb *VarBind) marshal(buf *bytes.Buffer) error {

	for _, v := range vb.entries {
		// type
		err := binary.Write(buf, binary.LittleEndian, &v.Type)
		if err != nil {
			return err
		}
		// reserved
		var l = uint16(0)
		err = binary.Write(buf, binary.LittleEndian, &l)
		if err != nil {
			return err
		}

		buf.Write([]uint8{uint8(len(v.OID)), 0, 0, 0})
		
		err = binary.Write(buf, binary.LittleEndian, v.OID)
		if err != nil {
			return err
		}
		
		if v.Data != nil {
			err = v.Data.MarshalAgentX(buf)
			if err != nil {
				return err
			}
		}
		
	}
	
	return nil
	
}

