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

type Counter64 struct {
	val		uint64
}

func (c *Counter64) Get() uint64 {
	return atomic.LoadUint64(&c.val)
}

func (c *Counter64) Set(v uint64) {
	atomic.StoreUint64(&c.val, v)
}

func (c *Counter64) Inc(v uint64) {
	atomic.AddUint64(&c.val, v)
}

func (c *Counter64) MarshalAgentX(buf *bytes.Buffer) error {
	v := atomic.LoadUint64(&c.val)
	return binary.Write(buf, binary.LittleEndian, v)
}

func (c *Counter64) UnmarshalAgentX(buf *bytes.Buffer) error {
	var v uint64
	err := binary.Read(buf, binary.LittleEndian, v)
	if err != nil { return err }
	c.Set(v)
	return nil
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
//	a := uint64(*c)
//	atomic.StoreUint64(&a, v)
}
