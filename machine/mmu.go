package machine

type Privilege int

const (
	KernelPrivilege Privilege = iota
	UserPrivilege
)

type Segment struct {
	Base     uint32
	Limit    uint32
	ReadOnly bool
	Present  bool
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
	Mode     MMUMode
	Segments map[int]Segment
}

func GetMMU() *MMU {
	return &MMU{
		Mode:     DirectMode,
		Segments: make(map[int]Segment),
	}
}

func (m *MMU) Translate(segID int, virtualAddress, acc AccessType) (uint32, error) {
	if m.Mode == DirectMode {
		return uint32(virtualAddress), nil
	}
	// ainda não implementado
	return uint32(0), nil
}
