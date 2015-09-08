package agentx

import (
	"fmt"
	"errors"
)

type agentXError uint16

const (
	ErrAgentXNoError			agentXError		= 0
	ErrAgentXopenFailed							= 256
	ErrAgentXnotOpen							= 257
	ErrAgentXindexWrongType						= 258
	ErrAgentXindexAlreadyAllocated				= 259
	ErrAgentXindexNoneAvailable					= 260
	ErrAgentXindexNotAllocated					= 261
	ErrAgentXunsupportedContext					= 262
	ErrAgentXduplicateRegistration				= 263
	ErrAgentXunknownRegistration				= 264
	ErrAgentXunknownAgentCaps					= 265
	ErrAgentXparseError							= 266
	ErrAgentXrequestDenied						= 267
	ErrAgentXprocessingError					= 268
)

var (
	ErrOpenFailed								= errors.New("Open failed.")
	ErrRegisterFailed							= errors.New("Register failed.")
)

func (e agentXError) Error() string {
	switch {
	case e == ErrAgentXNoError: return "No Error"
	case e == ErrAgentXopenFailed: return "Open Failed"
	case e == ErrAgentXnotOpen: return "Not Open"
	case e == ErrAgentXindexWrongType: return "Index is wrong type"
	case e == ErrAgentXindexAlreadyAllocated: return "Index is already allocated"
	case e == ErrAgentXindexNoneAvailable: return "Index has no entries available"
	case e == ErrAgentXindexNotAllocated: return "Index is not allocated"
	case e == ErrAgentXunsupportedContext: return "Context is unsupported"
	case e == ErrAgentXduplicateRegistration: return "Duplicate registration"
	case e == ErrAgentXunknownRegistration: return "Unknown registration"
	case e == ErrAgentXunknownAgentCaps: return "Unknown agent capabilities"
	case e == ErrAgentXparseError: return "Parse Error"
	case e == ErrAgentXrequestDenied: return "Request Denied"
	case e == ErrAgentXprocessingError: return "Processing Error"
	default: return fmt.Sprintf("Unknown error %d", e)
	}
}

