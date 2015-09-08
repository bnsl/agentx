package agentx

import (
	"fmt"
	"strings"
	"strconv"
	"log"
)

// OID reprents a numeric object ID.
type OID []uint32

// AsString formats the OID as a string.
func (o OID) String() string {
	q := make([]string, len(o))
	for i := range o {
		q[i] = fmt.Sprintf("%d", o[i])
	}
	return strings.Join(q, ".")
}

// sortString formats the OID as a string padded to 5x digits for each OID
func (o OID) sortString() string {
	q := make([]string, len(o))
	for i := range o {
		q[i] = fmt.Sprintf("%.5d", o[i])
	}
	return strings.Join(q, ".")
}

// HasPrefix answers the question "does this OID have this prefix?"
func (a OID) HasPrefix(b OID) bool {
	if len(a) < len(b) {
		return false
	}

	for i := 0; i < len(b); i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// IsEqualTo checks whether this OID == that OID.
func (a OID) IsEqualTo(b OID) bool {
	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// ComesBefore answers the question "does this OID sort before that OID?"
func (a OID) ComesBefore(b OID) bool {
	var size int
	if len(a) < len(b) {
		size = len(a)
	} else {
		size = len(b)
	}

	for i := 0; i < size; i++ {
		if a[i] < b[i] {
			return true
		}
		if a[i] > b[i] {
			return false
		}
	}

	if len(a) < len(b) {
		return true
	}

	return false
}

// Parse a string into an OID
func OIDFromString(s string) (OID, error) {
	t := strings.Split(s, ".")
	var o OID

	if s[0] == '.' {
		o = make(OID, len(t) - 1)
		for i, j := range t[1:] {
			log.Printf("i = %v, j = %v", i, j)
			k, err := strconv.Atoi(j)
			if err != nil {
				return nil, err
			}
			o[i] = uint32(k)
		}
	} else {
		o = make(OID, len(t))
		for i, j := range t {
			k, err := strconv.Atoi(j)
			if err != nil {
				return nil, err
			}
			o[i] = uint32(k)
		}
	}
	
	return o, nil
}
