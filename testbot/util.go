package testbot

import (
	"github.com/headzoo/surf/jar"
)

// NullHistory is a dummy github.com/headzoo/surf/jar.History that discards all history.
type NullHistory struct{}

func (h NullHistory) Len() int              { return 0 }
func (h NullHistory) Push(p *jar.State) int { return 0 }
func (h NullHistory) Pop() *jar.State       { return nil }
func (h NullHistory) Top() *jar.State       { return nil }
