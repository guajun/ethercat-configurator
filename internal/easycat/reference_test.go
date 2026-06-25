package easycat

import (
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/guajun/ethercat-configurator/internal/config"
	"github.com/guajun/ethercat-configurator/internal/model"
	"github.com/guajun/ethercat-configurator/internal/pdo"
)

var referenceCases = []referenceCase{
	{
		name:            "easycat_safe_64",
		devicePath:      "../../examples/easycat-safe-64/device.yaml",
		xmlPath:         "../../fixtures/easycat/easycat_safe_64/easycat_safe_64.xml",
		binPath:         "../../fixtures/easycat/easycat_safe_64/easycat_safe_64.bin",
		wantRXAddress:   0x1000,
		wantTXAddress:   0x1200,
		wantByteLength:  64,
		wantProductCode: 0xabcde064,
	},
	{
		name:            "easycat_fixed_192",
		devicePath:      "../../examples/easycat-reference/device.yaml",
		xmlPath:         "../../fixtures/easycat/ethercat_configurator_reference/ethercat_configurator_reference.xml",
		binPath:         "../../fixtures/easycat/ethercat_configurator_reference/ethercat_configurator_reference.bin",
		wantRXAddress:   0x1000,
		wantTXAddress:   0x1800,
		wantByteLength:  192,
		wantProductCode: 0xabcde010,
	},
}

type referenceCase struct {
	name            string
	devicePath      string
	xmlPath         string
	binPath         string
	wantRXAddress   uint32
	wantTXAddress   uint32
	wantByteLength  uint32
	wantProductCode uint32
}

const easyCATKnownBadXMLPath = "../../fixtures/easycat/ethercat_configurator_reference/known_bad_overlap.xml"

type etherCATInfo struct {
	Vendor       vendor       `xml:"Vendor"`
	Descriptions descriptions `xml:"Descriptions"`
}

type vendor struct {
	ID   string `xml:"Id"`
	Name string `xml:"Name"`
}

type descriptions struct {
	Devices devices `xml:"Devices"`
}

type devices struct {
	Device device `xml:"Device"`
}

type device struct {
	Type   deviceType    `xml:"Type"`
	Name   namedText     `xml:"Name"`
	SMs    []syncManager `xml:"Sm"`
	RxPDOs []easyCATPDO  `xml:"RxPdo"`
	TxPDOs []easyCATPDO  `xml:"TxPdo"`
}

type deviceType struct {
	ProductCode string `xml:"ProductCode,attr"`
	RevisionNo  string `xml:"RevisionNo,attr"`
	Value       string `xml:",chardata"`
}

type namedText struct {
	Value string `xml:",chardata"`
}

type syncManager struct {
	StartAddress string `xml:"StartAddress,attr"`
	Value        string `xml:",chardata"`
}

type easyCATPDO struct {
	Index   string         `xml:"Index"`
	Name    string         `xml:"Name"`
	Entries []easyCATEntry `xml:"Entry"`
}

type easyCATEntry struct {
	Index    string `xml:"Index"`
	Subindex uint8  `xml:"SubIndex"`
	BitLen   uint16 `xml:"BitLen"`
	Name     string `xml:"Name"`
	DataType string `xml:"DataType"`
}

type normalizedEntry struct {
	Direction  model.Direction
	Name       string
	BitLength  uint16
	BitOffset  uint32
	ByteOffset uint32
}

func TestEasyCATReferenceDeviceLayout(t *testing.T) {
	for _, testCase := range referenceCases {
		t.Run(testCase.name, func(t *testing.T) {
			loadResult := config.LoadFile(testCase.devicePath)
			if len(loadResult.Diagnostics) != 0 {
				t.Fatalf("expected valid EasyCAT reference device, got %#v", loadResult.Diagnostics)
			}
			layout, layoutDiagnostics := pdo.BuildLayout(loadResult.Device)
			addressMap, addressDiagnostics := pdo.BuildAddressMap(loadResult.Device, layout)
			if len(layoutDiagnostics) != 0 || len(addressDiagnostics) != 0 {
				t.Fatalf("expected valid EasyCAT reference layout, got layout=%#v address=%#v", layoutDiagnostics, addressDiagnostics)
			}
			if layout.RX.ByteLength != testCase.wantByteLength || layout.TX.ByteLength != testCase.wantByteLength {
				t.Fatalf("expected %d-byte RX/TX layouts, got RX=%d TX=%d", testCase.wantByteLength, layout.RX.ByteLength, layout.TX.ByteLength)
			}
			if addressMap.RX.Address != testCase.wantRXAddress || addressMap.TX.Address != testCase.wantTXAddress {
				t.Fatalf("expected EasyCAT PDI addresses 0x%04x/0x%04x, got RX=0x%04x TX=0x%04x", testCase.wantRXAddress, testCase.wantTXAddress, addressMap.RX.Address, addressMap.TX.Address)
			}
		})
	}
}

func TestEasyCATReferenceXMLMatchesDevice(t *testing.T) {
	for _, testCase := range referenceCases {
		t.Run(testCase.name, func(t *testing.T) {
			data, err := os.ReadFile(testCase.xmlPath)
			if os.IsNotExist(err) {
				t.Skipf("EasyCAT XML reference not generated yet: %s", testCase.xmlPath)
			}
			if err != nil {
				t.Fatalf("expected EasyCAT XML reference, got %v", err)
			}
			var info etherCATInfo
			if err := xml.Unmarshal(data, &info); err != nil {
				t.Fatalf("expected parseable EasyCAT XML, got %v", err)
			}

			loadResult := config.LoadFile(testCase.devicePath)
			layout, _ := pdo.BuildLayout(loadResult.Device)
			addressMap, _ := pdo.BuildAddressMap(loadResult.Device, layout)

			if normalizeHex(info.Vendor.ID, 8) != hex32(loadResult.Device.Identity.VendorID) {
				t.Fatalf("vendor mismatch: EasyCAT=%s device=%s", info.Vendor.ID, hex32(loadResult.Device.Identity.VendorID))
			}
			if normalizeHex(info.Descriptions.Devices.Device.Type.ProductCode, 8) != hex32(loadResult.Device.Identity.ProductCode) {
				t.Fatalf("product mismatch: EasyCAT=%s device=%s", info.Descriptions.Devices.Device.Type.ProductCode, hex32(loadResult.Device.Identity.ProductCode))
			}
			if normalizeHex(info.Descriptions.Devices.Device.Type.RevisionNo, 8) != hex32(loadResult.Device.Identity.Revision) {
				t.Fatalf("revision mismatch: EasyCAT=%s device=%s", info.Descriptions.Devices.Device.Type.RevisionNo, hex32(loadResult.Device.Identity.Revision))
			}
			assertSyncManager(t, info.Descriptions.Devices.Device.SMs, "Outputs", addressMap.RX.Address)
			assertSyncManager(t, info.Descriptions.Devices.Device.SMs, "Inputs", addressMap.TX.Address)
			assertEntries(t, model.DirectionRX, info.Descriptions.Devices.Device.RxPDOs, layout.RX)
			assertEntries(t, model.DirectionTX, info.Descriptions.Devices.Device.TxPDOs, layout.TX)
		})
	}
}

func TestEasyCATKnownBadXMLDetectsSyncManagerOverlap(t *testing.T) {
	data, err := os.ReadFile(easyCATKnownBadXMLPath)
	if os.IsNotExist(err) {
		t.Skipf("known-bad EasyCAT XML not present: %s", easyCATKnownBadXMLPath)
	}
	if err != nil {
		t.Fatalf("expected known-bad EasyCAT XML, got %v", err)
	}
	var info etherCATInfo
	if err := xml.Unmarshal(data, &info); err != nil {
		t.Fatalf("expected parseable known-bad EasyCAT XML, got %v", err)
	}
	rxStart, ok := syncManagerAddress(info.Descriptions.Devices.Device.SMs, "Outputs")
	if !ok {
		t.Fatalf("missing known-bad Outputs SM")
	}
	txStart, ok := syncManagerAddress(info.Descriptions.Devices.Device.SMs, "Inputs")
	if !ok {
		t.Fatalf("missing known-bad Inputs SM")
	}
	rxBytes := pdoByteLength(info.Descriptions.Devices.Device.RxPDOs)
	if rxStart+rxBytes*3 <= txStart {
		t.Fatalf("expected known-bad EasyCAT XML to overlap, got RX 0x%04x size=%d TX 0x%04x", rxStart, rxBytes, txStart)
	}
	overlapBytes := rxStart + rxBytes*3 - txStart
	if overlapBytes != 64 {
		t.Fatalf("expected 64-byte overlap, got %d bytes", overlapBytes)
	}
}

func TestEasyCATReferenceBinaryMatchesIdentity(t *testing.T) {
	for _, testCase := range referenceCases {
		t.Run(testCase.name, func(t *testing.T) {
			data, err := os.ReadFile(testCase.binPath)
			if os.IsNotExist(err) {
				t.Skipf("EasyCAT binary reference not generated yet: %s", testCase.binPath)
			} else if err != nil {
				t.Fatalf("expected EasyCAT binary reference, got %v", err)
			}
			if len(data) < 0x1c {
				t.Fatalf("EasyCAT binary is too short: %d bytes", len(data))
			}

			loadResult := config.LoadFile(testCase.devicePath)
			identity := loadResult.Device.Identity
			vendorID := littleEndianUint32(data[0x10:0x14])
			productCode := littleEndianUint32(data[0x14:0x18])
			revision := littleEndianUint32(data[0x18:0x1c])
			if vendorID != identity.VendorID || productCode != identity.ProductCode || revision != identity.Revision {
				t.Fatalf("EasyCAT binary identity mismatch: bin vendor=%s product=%s revision=%s device vendor=%s product=%s revision=%s", hex32(vendorID), hex32(productCode), hex32(revision), hex32(identity.VendorID), hex32(identity.ProductCode), hex32(identity.Revision))
			}
		})
	}
}

func assertSyncManager(t *testing.T, syncManagers []syncManager, name string, wantAddress uint32) {
	t.Helper()
	for _, syncManager := range syncManagers {
		if strings.TrimSpace(syncManager.Value) != name {
			continue
		}
		if normalizeHex(syncManager.StartAddress, 4) != hex16(uint16(wantAddress)) {
			t.Fatalf("%s SM address mismatch: EasyCAT=%s device=%s; 192-byte 3-buffer layouts need input at 0x1800 to avoid RX 0x1000..0x1240 overlapping TX 0x1200", name, syncManager.StartAddress, hex16(uint16(wantAddress)))
		}
		return
	}
	t.Fatalf("missing EasyCAT sync manager %q", name)
}

func syncManagerAddress(syncManagers []syncManager, name string) (uint32, bool) {
	for _, syncManager := range syncManagers {
		if strings.TrimSpace(syncManager.Value) != name {
			continue
		}
		address, err := parseHex(syncManager.StartAddress)
		if err != nil {
			return 0, false
		}
		return uint32(address), true
	}
	return 0, false
}

func pdoByteLength(pdos []easyCATPDO) uint32 {
	var bits uint32
	for _, pdo := range pdos {
		for _, entry := range pdo.Entries {
			bits += uint32(entry.BitLen)
		}
	}
	return (bits + 7) / 8
}

func assertEntries(t *testing.T, direction model.Direction, pdos []easyCATPDO, layout model.LayoutDirection) {
	t.Helper()
	var actual []normalizedEntry
	for _, easyCATPDO := range pdos {
		for _, entry := range easyCATPDO.Entries {
			actual = append(actual, normalizedEntry{Direction: direction, Name: strings.TrimSpace(entry.Name), BitLength: entry.BitLen})
		}
	}
	if len(actual) != len(layout.Entries) {
		t.Fatalf("%s PDO entry count mismatch: EasyCAT=%d device=%d", direction, len(actual), len(layout.Entries))
	}
	for index, entry := range layout.Entries {
		actual[index].BitOffset = entry.BitOffset
		actual[index].ByteOffset = entry.ByteOffset
		want := normalizedEntry{Direction: direction, Name: entry.Name, BitLength: entry.BitLength, BitOffset: entry.BitOffset, ByteOffset: entry.ByteOffset}
		if actual[index] != want {
			t.Fatalf("%s PDO entry %d mismatch: EasyCAT=%#v device=%#v", direction, index, actual[index], want)
		}
	}
}

func normalizeHex(value string, width int) string {
	number, err := parseHex(value)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(value))
	}
	return fmt.Sprintf("0x%0*x", width, number)
}

func parseHex(value string) (uint64, error) {
	clean := strings.TrimSpace(value)
	clean = strings.TrimPrefix(clean, "#x")
	clean = strings.TrimPrefix(clean, "#X")
	clean = strings.TrimPrefix(clean, "0x")
	clean = strings.TrimPrefix(clean, "0X")
	return strconv.ParseUint(clean, 16, 64)
}

func hex16(value uint16) string {
	return fmt.Sprintf("0x%04x", value)
}

func hex32(value uint32) string {
	return fmt.Sprintf("0x%08x", value)
}

func littleEndianUint32(data []byte) uint32 {
	return uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
}
