package machine

import (
	"fmt"
)

type Privilege int
const (
	KernelPrivilege Privilege = iota
	UserPrivilege
)

type Segment struct {
	Base uint32
	Limit uint32
	ReadOnly bool
	Present bool
}

type MMUMode int
const (
	DirectMode MMUMode = iota
	SegmentedMode
)

type AccessType int
const (
	AccessRead AccessType = iota
	AccessWrite
)

type MMU struct {
	Mode MMUMode
	Segments map[int]Segment
}
