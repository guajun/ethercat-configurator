package model

type Direction string

type BufferMode string

const (
	DirectionRX Direction = "rx"
	DirectionTX Direction = "tx"
)

const (
	BufferModeSingle BufferMode = "single"
	BufferModeDouble BufferMode = "double"
	BufferModeTriple BufferMode = "triple"
)

type Device struct {
	Identity         Identity
	ProcessData      ProcessData
	ObjectDictionary []ObjectEntry
	PDOs             PDOSet
}

type Identity struct {
	VendorID    uint32
	ProductCode uint32
	Revision    uint32
	Name        string
}

type ProcessData struct {
	RX ProcessDataRegion
	TX ProcessDataRegion
}

type ProcessDataRegion struct {
	Address    uint32
	Size       uint32
	BufferMode BufferMode
	Configured bool
}

type ObjectEntry struct {
	Index        uint16
	Subindex     uint8
	Name         string
	Type         string
	BitLength    uint16
	Access       string
	DefaultValue string
	PDO          Direction
}

type PDOSet struct {
	RX []PDO
	TX []PDO
}

type PDO struct {
	Index   uint16
	Name    string
	Entries []PDOEntry
}

type PDOEntry struct {
	Index    uint16
	Subindex uint8
}

type Layout struct {
	RX LayoutDirection
	TX LayoutDirection
}

type LayoutDirection struct {
	Direction   Direction
	Entries     []LayoutEntry
	BitLength   uint32
	ByteLength  uint32
	PaddingBits uint8
}

type LayoutEntry struct {
	Index      uint16
	Subindex   uint8
	Name       string
	Type       string
	BitOffset  uint32
	ByteOffset uint32
	BitLength  uint16
	ByteLength uint32
	PDOIndex   uint16
}

type AddressMap struct {
	RX AddressRegion
	TX AddressRegion
}

type AddressRegion struct {
	Direction     Direction
	Address       uint32
	RequiredBytes uint32
	BufferBytes   uint32
	BufferMode    BufferMode
	End           uint32
	FromDefault   bool
}

func BufferCount(mode BufferMode) uint32 {
	switch mode {
	case BufferModeDouble:
		return 2
	case BufferModeTriple:
		return 3
	default:
		return 1
	}
}

func ObjectKey(index uint16, subindex uint8) uint32 {
	return uint32(index)<<8 | uint32(subindex)
}
