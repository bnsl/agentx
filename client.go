package agentx

import (
	"net"
	"log"
	"bytes"
	"math"
	"sync"
	"time"
)

type clientRequest struct {
	pkttype			AgentXPDUType
	payload			*bytes.Buffer
	replyChan		chan interface{}
}

type Client struct {
	started		time.Time
	conn		net.Conn
	oid			*OID
	submit		chan clientRequest
	PacketID	uint32
	SessionID	uint32
	resultTo	map[uint32]chan interface{}
	rwlock		sync.RWMutex
	trees		map[string]*OIDTree
	
	closing		bool
	rxDied		chan bool
	plsDie		chan bool
	Closed		chan bool
}

func (c *Client) loop() {

	go func() {
		for {
			buf := make([]byte, 1500)
			l, err := c.conn.Read(buf)
//			log.Printf("RX: %d bytes", l)
//			log.Printf("RX: % v", buf[0:l])
			if err != nil {
				if !c.closing {
					log.Printf("RX chan exits, sadfaec: %v", err)
				}
				close(c.rxDied)
				return
			}
			h, err := unmarshalHeader(buf)
			if err != nil {
				log.Printf("Reply error: %v", err)
			}
			switch {
			case h.Type == agentx_Get:
				go func() {
					r, err := unmarshalGet(buf[20:l])
					if err != nil {
						log.Printf("ERROR: %v", err)
						return
					}

					vb := NewVarBind()
				
					for _, l := range r.OIDs {
						var t *OIDTree
						var ok bool
						var o *OIDEntry
				
						c.rwlock.RLock()
						for i := len(l); i > 6; i-- {
							if t, ok = c.trees[l[:i].String()]; ok {
								break
							}
						}

						c.rwlock.RUnlock()

						if (t != nil) {
							o, err = t.Get(l)
						}
					
						if t == nil || o == nil {
							vb.Add(vbnoSuchObject, l, nil)
							continue
						}
						
						vb.Add(o.Value.Type(), l, o.Value)
					}
				
					c.sendResponse(*h, vb)
				}()
			case h.Type == agentx_GetNext:
				go func() {
					r, err := unmarshalGetNext(buf[20:l])
					if err != nil {
						log.Printf("ERROR: %v", err)
						return
					}

					vb := NewVarBind()
				
					for _, l := range r.OIDs {
						var t *OIDTree
						var ok bool
						var o *OIDEntry
				
						c.rwlock.RLock()
						for i := len(l); i > 6; i-- {
							if t, ok = c.trees[l[:i].String()]; ok {
								break
							}
						}
						c.rwlock.RUnlock()

						if (t != nil) {
							o, err = t.Next(l)
						}
					
						if t == nil || o == nil {
							vb.Add(vbendOfMibView, l, nil)
							continue
						}
						
						vb.Add(o.Value.Type(), o.OID, o.Value)
					}
				
					c.sendResponse(*h, vb)
				}()
			case h.Type == agentx_Response:
				c.rwlock.Lock()
				rep, ok := c.resultTo[h.PacketID]
				if ok {
					delete(c.resultTo, h.PacketID)
				}
				c.rwlock.Unlock()
				if ok {
					defer close(rep)
					if err != nil {
						rep <- err
					} else {
						r, err := unmarshalResponse(buf[20:l])
						r.header = *h
						if err != nil {
							rep <- err
							continue
						}
						rep <- r
						continue
					}
				} else {
					log.Printf("Reply for unknown request")
				}
			default:
				log.Printf("Unknown packet type recieved %d", h.Type)
			}
		}
	}()

	for {
		select {
		case r := <- c.submit:
			var h Header
			c.PacketID++
			c.resultTo[c.PacketID] = r.replyChan
			h.PacketID = c.PacketID
			h.SessionID = c.SessionID
			h.Version = 1
			h.TransactionID = 0
			h.Type = r.pkttype
			newlen := int(math.Ceil(float64(r.payload.Len())/4.0) * 4)

			for i := r.payload.Len(); i < newlen; i++ {
				r.payload.Write([]byte{0})
			}
			h.Length = uint32(newlen)

			b := new(bytes.Buffer)
			h.Marshal(b)
			b.Write(r.payload.Bytes())
			l, err := c.conn.Write(b.Bytes())
			if err != nil {
				log.Printf("Error: %v", err)
			}
			if l != b.Len() {
				log.Printf("Error, only send %d of %d bytes", l, b.Len)
			}
		case _, open := <- c.plsDie:
			if !open {
				c.conn.Close()
				close(c.Closed)
				return
			}
		case _, open := <- c.rxDied:
			if !open {
				c.conn.Close()
				close(c.Closed)
				return
			}
		}
	}
}

func (c *Client) send(ty AgentXPDUType, buf *bytes.Buffer) (interface{}, error) {
	mychan := make(chan interface{})
	
	c.submit <- clientRequest{ty, buf, mychan}
	
	rep := <- mychan

	if err, ok := rep.(error); ok {
		log.Printf("Error: %v", err)
		return nil, err
	}

	return rep, nil
}

func (c *Client) sendResponse(h Header, vb *VarBind) {
	payload := new(bytes.Buffer)
	
	h.Type = agentx_Response
	
	pdu := agentx_Response_PDU{h, uint32(time.Since(c.started).Seconds()), ErrAgentXNoError, 0, vb}
	pdu.marshal(payload)

	newlen := int(math.Ceil(float64(payload.Len())/4.0) * 4)

	for i := payload.Len(); i < newlen; i++ {
		payload.Write([]byte{0})
	}
	h.Length = uint32(newlen)

	b := new(bytes.Buffer)
	h.Marshal(b)
	b.Write(payload.Bytes())
	l, err := c.conn.Write(b.Bytes())
	if err != nil {
		log.Printf("Error: %v", err)
	}
	if l != b.Len() {
		log.Printf("Error, only sent %d bytes of %d", l, b.Len())
	}
}

func (c *Client) Close() {
	c.closing = true

	req := agentx_Close_PDU{}

	req.reason = agentxCloseReason_Shutdown
	
	b, err := req.Marshal()
	
	rep, err := c.send(agentx_Close, b)
	if err != nil {
		log.Printf("Error closing: %v", err)
		return
	}

	if _, ok := rep.(*agentx_Response_PDU); !ok {
		log.Printf("Type passed back not agentx_Response_PDU or error")
	}

	close(c.plsDie)
	
	for {
		select {
		case _, open := <- c.Closed:
			if !open {
				return
			}
		}
	}
}

func (c *Client) sendOpen() error {

	h := agentx_Open_PDU{}
	h.Timeout = 60
	h.desc = "AgentX-GO/0.0"

	buf, err := h.Marshal()
	if err != nil {
		return err
	}

	rep, err := c.send(agentx_Open, buf)
	if err != nil {
		return err
	}

	if res, ok := rep.(*agentx_Response_PDU); ok {
		c.SessionID = res.header.SessionID
	} else {
		log.Printf("Type passed back not agentx_Response_PDU or error")
		c.conn.Close()
		return ErrOpenFailed
	}

	return nil
}

func (c *Client) Register(base OID, oidtree *OIDTree) error {
	
	req := agentx_Register_PDU{OID: base}

	buf, err := req.Marshal()
	if err != nil {
		return err
	}

	rep, err := c.send(agentx_Register, buf)
	if err != nil {
		return err
	}

	if res, ok := rep.(*agentx_Response_PDU); ok {
		c.SessionID = res.header.SessionID
	} else {
		log.Printf("Type passed back not agentx_Response_PDU or error")
		return ErrRegisterFailed
	}
	
	c.rwlock.Lock()
	defer c.rwlock.Unlock()
	
	c.trees[base.String()] = oidtree
	
	return nil
}

func newClient() (*Client) {
	return &Client{submit: make(chan clientRequest), resultTo: make(map[uint32]chan interface{}), plsDie: make(chan bool), Closed: make(chan bool), rxDied: make(chan bool), trees: make(map[string]*OIDTree), started: time.Now()}
}

func ConnectAndServe(netw string, address string, oid *OID) (*Client, error) {
	conn, err := net.Dial(netw, address)
	if err != nil {
		return nil, err
	}
	
	client := newClient()
	client.conn = conn

	go client.loop()

	err = client.sendOpen()
	if err != nil {
		return nil, err
	}

	return client, nil
}
