package state

import (
	"time"

	"github.com/google/go-github/v33/github"
)

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
	Name            string
	LastValue       []*github.WorkflowRun
	LastRequest     time.Time
	Status          Status
	NextForcedScale *time.Time
}
