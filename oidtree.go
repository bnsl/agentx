package agentx

import (
	"sync"
	"sort"
	"errors"
	"log"
)

var (
	ErrOIDNotFound = errors.New("Could not find OID in tree")
	ErrNoNextOID = errors.New("There is not another OID in the tree")
)

type OIDEntry struct {
	OID			OID
	Value		SNMPValue
	Next		*OIDEntry
	Tree		*OIDTree
	sortString	string
}

type OIDTree struct {
	lock		sync.RWMutex
	Subtree		map[string]*OIDEntry
}

func NewOIDTree() *OIDTree {
	return &OIDTree{Subtree: make(map[string]*OIDEntry)}
}

func (ot *OIDTree) NewEntry(o OID, value SNMPValue) *OIDEntry {
	e := &OIDEntry{OID: o, Value: value, sortString: o.sortString(), Tree: ot}
	ot.Subtree[o.sortString()] = e

	ot.Refine()

	return e
}

// this sorts the tree and shit
func (ot *OIDTree) Refine() {
	if len(ot.Subtree) < 2 { return }
	k := make(sort.StringSlice, len(ot.Subtree))
	i := 0
	for _, v := range ot.Subtree {
		k[i] = v.sortString
		i++
	}
	k.Sort()
	for i, v := range k {
//		fmt.Printf("i == %d\n", i)
		if i < len(k)-1 {
			ot.Subtree[v].Next = ot.Subtree[k[i+1]]
		}
		if i == len(k)-1 {
			ot.Subtree[v].Next = nil
		}
//		fmt.Printf("%d %s\n", i, v)
	}
}

func (ot *OIDTree) Get(o OID) (*OIDEntry, error) {
	if e, ok := ot.Subtree[o.sortString()]; ok {
		return e, nil
	}
	return nil, ErrOIDNotFound
}

func (ot *OIDTree) Next(o OID) (*OIDEntry, error) {
	log.Printf("NEXT")
	if e, ok := ot.Subtree[o.sortString()]; ok {
		log.Printf("Couldn't find a tree?")
		if e.Next != nil {
			return e.Next, nil
		}
		return nil, ErrNoNextOID
	}
	log.Printf("HELLO")
	k := make(sort.StringSlice, len(ot.Subtree)+1)
	x := o.sortString()
	k[0] = x
	i := 1
	for _, v := range ot.Subtree {
		k[i] = v.sortString
		i++
	}
	k.Sort()
	i = 0
	for _, v := range k {
		if v == x {
			log.Printf("%s == %s ?", v, x)
			if i == len(k) - 1 {
				return nil, ErrNoNextOID
			}
			return ot.Subtree[k[i+1]], nil
		}
		i++
	}
	
	return nil, ErrOIDNotFound
}
