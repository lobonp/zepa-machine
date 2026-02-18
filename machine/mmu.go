package machine

import "fmt"

const (
	MaxSegmentSizeBits = 12
	MaxSegmentSize     = 1 << MaxSegmentSizeBits
	OffsetMask         = MaxSegmentSize - 1
	SegmentShift       = MaxSegmentSizeBits
	NumSegments        = 4
)

type Privilege int

const (
	KernelPrivilege Privilege = iota
	UserPrivilege
)

type MMUMode int

const (
	ModeFlat MMUMode = iota
	ModeSegmented
)

type AccessType int

const (
	Execute AccessType = 1 << iota
	Write
	Read
)

type Segment struct {
	Base          uint32
	Limit         uint32
	GrowsPositive bool
	Protection    AccessType
	Priv          Privilege
}

type MMU struct {
	Mode     MMUMode
	Segments [NumSegments]Segment
}

func (m *MMU) Translate(virtualAddress uint32, access AccessType, currentPriv Privilege) (uint32, error) {
	var physicalAddress uint32
	err := fmt.Errorf("SEGMENTATION_FAULT: Falha na tradução")

	if m.Mode == ModeFlat {
		return virtualAddress, nil
	}

	segmentID := virtualAddress >> SegmentShift
	offset := virtualAddress & OffsetMask

	if segmentID >= uint32(len(m.Segments)) {
		return 0, fmt.Errorf("SEGMENTATION_FAULT: ID %d inválido", segmentID)
	}

	seg := m.Segments[segmentID]

	if currentPriv > seg.Priv {
		return 0, fmt.Errorf("SEGMENTATION_FAULT: Privilege violation")
	}

	if seg.Protection&access == 0 {
		err = fmt.Errorf("SEGMENTATION_FAULT: Forbidden")
		return 0, err
	}

	if seg.GrowsPositive && offset < seg.Limit {
		physicalAddress = seg.Base + offset
		err = nil
	}

	if !seg.GrowsPositive {
		realOffset := int32(offset) - int32(MaxSegmentSize)
		if uint32(-realOffset) <= seg.Limit {
			physicalAddress = uint32(int32(seg.Base) + realOffset)
			err = nil
		}
	}

	return physicalAddress, err
}
