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
	rwlock		sync.Mutex
	stop		chan bool
	trees		map[string]*OIDTree
}

func (c *Client) loop() {

	go func() {
		for {
			buf := make([]byte, 1500)
			l, err := c.conn.Read(buf)
			log.Printf("RX: %d bytes", l)
			log.Printf("RX: % v", buf[0:l])
			if err != nil {
				log.Printf("RX chan exits, sadfaec: %v", err)
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
					log.Printf("GET: %v", r)

					vb := NewVarBind()
				
					for _, l := range r.OIDs {
						var t *OIDTree
						var ok bool
						var o *OIDEntry
				
						log.Printf("check for: %v", l)
						for i := len(l); i > 6; i-- {
							if t, ok = c.trees[l[:i].String()]; ok {
								break
							}
						}
						log.Printf("t := %v", t)
						if (t != nil) {
							o, err = t.Get(l)
						}
					
						if t == nil || o == nil {
							vb.Add(vbnoSuchObject, l, nil)
							continue
						}
						
						vb.Add(o.Value.Type(), l, o.Value)
					}
				
					log.Printf("vb == %v", vb.entries)

					c.sendResponse(*h, vb)
				}()
			case h.Type == agentx_GetNext:
				go func() {
					log.Printf("GET NEXT")
					r, err := unmarshalGetNext(buf[20:l])
					if err != nil {
						log.Printf("ERROR: %v", err)
						return
					}
					log.Printf("GET: %v", r)

					vb := NewVarBind()
				
					for _, l := range r.OIDs {
						var t *OIDTree
						var ok bool
						var o *OIDEntry
				
						log.Printf("check for: %v", l)
						for i := len(l); i > 6; i-- {
							if t, ok = c.trees[l[:i].String()]; ok {
								break
							}
						}
						log.Printf("t := %v", t)
						if (t != nil) {
							o, err = t.Next(l)
						}
					
						if t == nil || o == nil {
							vb.Add(vbendOfMibView, l, nil)
							continue
						}
						
						vb.Add(o.Value.Type(), o.OID, o.Value)
					}
				
					log.Printf("vb == %v", vb.entries)

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
						log.Printf("?")
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
			log.Printf("newlen: %d / %d", r.payload.Len(), newlen)

			for i := r.payload.Len(); i < newlen; i++ {
				r.payload.Write([]byte{0})
			}
			h.Length = uint32(newlen)

			b := new(bytes.Buffer)
			h.Marshal(b)
			b.Write(r.payload.Bytes())
			log.Printf("TX: % v", b.Bytes())
			l, err := c.conn.Write(b.Bytes())
			if err != nil {
				log.Printf("Error: %v", err)
			}
			log.Printf("TX bytes sent: %d", l)
		case _, open := <- c.stop:
			if !open {
				c.conn.Close()
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

	log.Printf("r: % v", rep)
	
	return rep, nil
}

func (c *Client) sendResponse(h Header, vb *VarBind) {
	payload := new(bytes.Buffer)
	
	h.Type = agentx_Response
	
	pdu := agentx_Response_PDU{h, uint32(time.Since(c.started).Seconds()), ErrAgentXNoError, 0, vb}
	pdu.marshal(payload)

	newlen := int(math.Ceil(float64(payload.Len())/4.0) * 4)
	log.Printf("newlen: %d / %d", payload.Len(), newlen)

	for i := payload.Len(); i < newlen; i++ {
		payload.Write([]byte{0})
	}
	h.Length = uint32(newlen)

	b := new(bytes.Buffer)
	h.Marshal(b)
	b.Write(payload.Bytes())
	log.Printf("TX: % v", b.Bytes())
	l, err := c.conn.Write(b.Bytes())
	if err != nil {
		log.Printf("Error: %v", err)
	}
	log.Printf("TX bytes sent: %d", l)
	
}

func (c *Client) Close() {
	req := agentx_Close_PDU{}

	req.reason = agentxCloseReason_Shutdown
	
	b, err := req.Marshal()
	
	rep, err := c.send(agentx_Close, b)
	if err != nil {
		log.Printf("Error closing: %v", err)
		return
	}

	if res, ok := rep.(*agentx_Response_PDU); ok {
		log.Printf("Got response: %v", res)
	} else {
		log.Printf("Type passed back not agentx_Response_PDU or error")
	}

	close(c.stop)
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
		log.Printf("Got response: %v", res)
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
		log.Printf("Got response: %v", res)
		c.SessionID = res.header.SessionID
	} else {
		log.Printf("Type passed back not agentx_Response_PDU or error")
		c.conn.Close()
		return ErrOpenFailed
	}
	
	c.rwlock.Lock()
	defer c.rwlock.Unlock()
	
	c.trees[base.String()] = oidtree
	
	return nil
}

func newClient() (*Client) {
	return &Client{submit: make(chan clientRequest), resultTo: make(map[uint32]chan interface{}), stop: make(chan bool), trees: make(map[string]*OIDTree), started: time.Now()}
}

func ConnectAndServe(netw string, address string, oid *OID) (*Client, error) {
	conn, err := net.Dial(netw, address)
	if err != nil {
		return nil, err
	}
	
	client := newClient()
	client.conn = conn

	go client.loop()

	log.Printf("Sending Open")
	err = client.sendOpen()
	if err != nil {
		return nil, err
	}

	return client, nil
}
