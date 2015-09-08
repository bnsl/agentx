package agentx

import (
	"sync/atomic"
	"bytes"
	"encoding/binary"
)

type SNMPValue interface {
	MarshalAgentX(buf *bytes.Buffer) error
	UnmarshalAgentX(buf *bytes.Buffer) error
	Type() varBindType
}

type Counter64 uint64

func (c *Counter64) Get() uint64 {
	return uint64(*c)
}

func (c *Counter64) Set(v uint64) {
	*c = Counter64(v)
}

func (c *Counter64) Inc(v uint64) {
	*c = *c + Counter64(v)
}

func (c *Counter64) MarshalAgentX(buf *bytes.Buffer) error {
	return binary.Write(buf, binary.LittleEndian, c)
}

func (c *Counter64) UnmarshalAgentX(buf *bytes.Buffer) error {
	return binary.Read(buf, binary.LittleEndian, c)
}

func (c *Counter64) Type() varBindType {
	return vbCounter64
}

// ----------------------------------------------------------------------------

type Gauge64 uint64

func (c *Gauge64) Get() uint64 {
	return uint64(*c)
}

func (c *Gauge64) Set(v uint64) {
	a := uint64(*c)
	atomic.StoreUint64(&a, v)
}
