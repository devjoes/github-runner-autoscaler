package state

import "time"

type Status int8

const (
	Unset Status = iota
	Valid
	Errored
)

func NewClientState(name string) *ClientState {
	return &ClientState{
		Name:   name,
		Status: Unset,
	}
}

type ClientState struct {
	Name        string
	LastValue   map[int64]map[string]string
	LastRequest time.Time
	Status      Status
}
