package protocol

import (
	"fmt"

	"github.com/mochi-co/hanami"
)

type keepAliveMsg struct {
	V  float64
	RV float64
	FS float64
	M  int
	S  int
}

func ParseKeepAlive(msg hanami.Msg) (*keepAliveMsg, error) {
	keepalive := &keepAliveMsg{
		V:  0,
		RV: 0,
		FS: 0,
		M:  0,
		S:  0,
	}
	if v, ok := msg["v"].(float64); !ok {
		return nil, fmt.Errorf("could not parse 'v' from keepalive")
	} else {
		keepalive.V = v
	}

	if rv, ok := msg["rv"].(float64); !ok {
		return nil, fmt.Errorf("could not parse 'rv' from keepalive")
	} else {
		keepalive.RV = rv
	}

	if fs, ok := msg["fs"].(float64); !ok {
		return nil, fmt.Errorf("could not parse 'fs' from keepalive")
	} else {
		keepalive.FS = fs
	}
	if m, ok := msg["m"].(float64); !ok {
		return nil, fmt.Errorf("could not parse 'm' from keepalive")
	} else {
		keepalive.M = int(m)
	}

	if s, ok := msg["s"].(float64); !ok {
		return nil, fmt.Errorf("could not parse 's' from keepalive")
	} else {
		keepalive.S = int(s)
	}

	return keepalive, nil
}
