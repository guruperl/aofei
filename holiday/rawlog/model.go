package rawlog

// after retrieving from database
// total, ym are int
// dhm, last are int16

import (
	"github.com/genelet/taodbi"
)

type Model struct {
	taodbi.Smodel
}
