package openflow

import (
	"fmt"
	"strings"

	"github.com/contiv/libOpenflow/openflow13"
	"github.com/contiv/ofnet/ofctrl"
)

type FlowStates struct {
	TableID         uint8
	PacketCount     uint64
	DurationNSecond uint32
}

type ofFlow struct {
	table *ofTable
	ofctrl.Flow

	// matchers is string slice, it is used to generate a readable match string of the Flow.
	matchers []string
	// protocol adds a readable protocol type in the match string of ofFlow.
	protocol Protocol
	// ctStateString is a temporary variable for the readable ct_state configuration. Its value is changed when the client
	// updates the matching condition of "ct_states". When FlowBuilder.Done is called, its value is added into the matchers.
	ctStateString string
	// ctStates is a temporary variable to maintain openflow13.CTStates. When FlowBuilder.Done is called, it is used to
	// set the CtStates field in ofctrl.Flow.Match.
	ctStates *openflow13.CTStates
	// lastAction is used to set ofctrl.Flow nextElem field. It is the last action of the Flow.
	lastAction ofctrl.FgraphElem
}

func (f *ofFlow) Reset() {
	f.Flow.Table = f.table.Table
}

func (f *ofFlow) Add() error {
	f.Flow.UpdateInstallStatus(false)
	err := f.Flow.Next(f.lastAction)
	if err != nil {
		return err
	}
	f.table.UpdateStatus(1)
	return nil
}

func (f *ofFlow) Modify() error {
	f.Flow.UpdateInstallStatus(true)
	err := f.Flow.Next(f.lastAction)
	if err != nil {
		return err
	}
	f.table.UpdateStatus(0)
	return nil
}

func (f *ofFlow) Delete() error {
	f.Flow.UpdateInstallStatus(true)
	err := f.Flow.Delete()
	if err != nil {
		return err
	}
	f.table.UpdateStatus(-1)
	return nil
}

func (f *ofFlow) Type() EntryType {
	return FlowEntry
}

func (f *ofFlow) KeyString() string {
	return f.MatchString()
}

func (f *ofFlow) MatchString() string {
	repr := fmt.Sprintf("table=%d,priority=%d", f.table.GetID(), f.Flow.Match.Priority)
	if f.protocol != "" {
		repr = fmt.Sprintf("%s,%s", repr, f.protocol)
	}

	if len(f.matchers) > 0 {
		repr += fmt.Sprintf(",%s", strings.Join(f.matchers, ","))
	}
	return repr
}

// CopyToBuilder returns a new FlowBuilder that copies the table, protocols, matches and CookieID of the Flow, but does
// not copy the actions, lastAction and other private status fields of the ofctrl.Flow, e.g., "realized" and "isInstalled".
func (f *ofFlow) CopyToBuilder() FlowBuilder {
	newFlow := ofFlow{
		table: f.table,
		Flow: ofctrl.Flow{
			Table:      f.Flow.Table,
			CookieID:   f.Flow.CookieID,
			CookieMask: f.Flow.CookieMask,
			Match:      f.Flow.Match,
		},
		matchers: f.matchers,
		protocol: f.protocol,
	}
	return &ofFlowBuilder{newFlow}
}

func (r *Range) ToNXRange() *openflow13.NXRange {
	return openflow13.NewNXRange(int(r[0]), int(r[1]))
}

func (r *Range) length() uint32 {
	return r[1] - r[0] + 1
}
