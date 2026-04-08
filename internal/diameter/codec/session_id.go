package codec

import (
	"fmt"
	"sync/atomic"
)

var sessionCounter atomic.Uint64

// NewSessionID returns a process-unique Diameter Session-Id for the given origin host.
func NewSessionID(originHost string) string {
	return fmt.Sprintf("%s;%d;%d", originHost, OriginStateID(), sessionCounter.Add(1))
}
