package machine

import "fmt"

type Privilege int

const (
	Ring0 Privilege = 0 // kernel
	Ring3 Privilege = 3 // user
)

type Segment struct {
	Base    uint32
	Priv    Privilege
	Limit   uint32
	Present bool
}

type MMUMode int

const (
	ModeFlat MMUMode = iota
	ModeSegmented
)

type AccessType int

const (
	AccessFetch AccessType = iota
	AccessRead
	AccessWrite
)

type MMU struct {
	Mode     MMUMode
	Segments [3]Segment
}

func (m *MMU) Translate(virtualAddress uint32) (uint32, error) {
	if m.Mode == ModeFlat {
		physicalAddress := virtualAddress
		return physicalAddress, nil
	}

	segmentID := virtualAddress >> 12
	offset := virtualAddress & 0x0FFF

	if segmentID >= uint32(len(m.Segments)) {
		return 0, fmt.Errorf("SEGMENTATION_FAULT: Invalid Segment")
	}

	segment := m.Segments[segmentID]

	if offset >= segment.Limit {
		return 0, fmt.Errorf("SEGMENTATION_FAULT: Out of bounds")
	}

	physicalAddress := segment.Base + offset
	return physicalAddress, nil
}
