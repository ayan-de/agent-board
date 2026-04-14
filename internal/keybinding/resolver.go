package keybinding

import (
	"strconv"
	"strings"
	"unicode"
)

type Resolver struct {
	keyMap      KeyMap
	chordMode   bool
	chordPrefix string
	digitBuf    strings.Builder
}

func NewResolver(km KeyMap) *Resolver {
	return &Resolver{keyMap: km}
}

func (r *Resolver) Resolve(key string) (Action, int) {
	if r.chordMode {
		if len(key) == 1 && unicode.IsDigit(rune(key[0])) {
			r.digitBuf.WriteString(key)
			n, _ := strconv.Atoi(r.digitBuf.String())
			return ActionGoToTicket, n
		}

		r.chordMode = false
		r.chordPrefix = ""
		r.digitBuf.Reset()
	}

	binding, ok := r.keyMap.Lookup(key)
	if !ok {
		return ActionNone, 0
	}

	if binding.IsChord {
		r.chordMode = true
		r.chordPrefix = key
		return ActionNone, 0
	}

	return binding.Action, 0
}

func (r *Resolver) Reset() {
	r.chordMode = false
	r.chordPrefix = ""
	r.digitBuf.Reset()
}

func (r *Resolver) InChordMode() bool {
	return r.chordMode
}
