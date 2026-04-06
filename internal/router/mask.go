package router

import (
	"strings"

	"github.com/harshithl1777/flock/internal/protocol"
)

// MethodMask stores the allowed methods for a route as a bitset.
type MethodMask uint16

const (
	AllowGet MethodMask = 1 << iota
	AllowHead
	AllowPost
	AllowPut
	AllowDelete
	AllowOptions
	AllowPatch
)

// Allows reports whether the mask permits the given method. HEAD is treated as
// allowed whenever GET is present.
func (m MethodMask) Allows(method protocol.Method) bool {
	if method == protocol.Head && m&AllowGet != 0 {
		return true
	}

	bit := methodBit(method)
	return bit != 0 && m&bit != 0
}

// newMethodMask converts configured methods into the runtime bitmask form.
func newMethodMask(methods []protocol.Method) MethodMask {
	var mask MethodMask

	for _, method := range methods {
		mask |= methodBit(method)
	}

	return mask
}

// AllowHeader returns the preformatted Allow header value for the method mask.
func (m MethodMask) AllowHeader() string {
	methods := make([]string, 0, 7)

	if m&AllowGet != 0 {
		methods = append(methods, string(protocol.Get))
	}
	if m&AllowHead != 0 || m&AllowGet != 0 {
		methods = append(methods, string(protocol.Head))
	}
	if m&AllowPost != 0 {
		methods = append(methods, string(protocol.Post))
	}
	if m&AllowPut != 0 {
		methods = append(methods, string(protocol.Put))
	}
	if m&AllowDelete != 0 {
		methods = append(methods, string(protocol.Delete))
	}
	if m&AllowOptions != 0 {
		methods = append(methods, string(protocol.Options))
	}
	if m&AllowPatch != 0 {
		methods = append(methods, string(protocol.Patch))
	}

	return strings.Join(methods, ", ")
}

// methodBit maps a protocol method to its corresponding bitmask flag.
func methodBit(method protocol.Method) MethodMask {
	switch method {
	case protocol.Get:
		return AllowGet
	case protocol.Head:
		return AllowHead
	case protocol.Post:
		return AllowPost
	case protocol.Put:
		return AllowPut
	case protocol.Delete:
		return AllowDelete
	case protocol.Options:
		return AllowOptions
	case protocol.Patch:
		return AllowPatch
	default:
		return 0
	}
}
