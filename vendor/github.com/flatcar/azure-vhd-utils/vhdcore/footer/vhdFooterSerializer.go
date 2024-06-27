package footer

import (
	"github.com/flatcar/azure-vhd-utils/vhdcore"
	"github.com/flatcar/azure-vhd-utils/vhdcore/writer"
)

// SerializeFooter returns the given VhdFooter instance as byte slice of length 512 bytes.
func SerializeFooter(footer *Footer) []byte {
	buffer := make([]byte, vhdcore.VhdFooterSize)
	writer := writer.NewVhdWriterFromByteSlice(buffer)

	writer.WriteBytesAt(0, footer.Cookie.Data)
	writer.WriteUInt32At(8, uint32(footer.Features))
	writer.WriteUInt32At(12, uint32(footer.FileFormatVersion))
	writer.WriteInt64At(16, footer.HeaderOffset)
	writer.WriteTimeStampAt(24, footer.TimeStamp)
	creatorApp := make([]byte, 4)
	copy(creatorApp, footer.CreatorApplication)
	writer.WriteBytesAt(28, creatorApp)
	writer.WriteUInt32At(32, uint32(footer.CreatorVersion))
	writer.WriteUInt32At(36, uint32(footer.CreatorHostOsType))
	writer.WriteInt64At(40, footer.PhysicalSize)
	writer.WriteInt64At(48, footer.VirtualSize)
	// + DiskGeometry
	writer.WriteUInt16At(56, footer.DiskGeometry.Cylinder)
	writer.WriteByteAt(58, footer.DiskGeometry.Heads)
	writer.WriteByteAt(59, footer.DiskGeometry.Sectors)
	// - DiskGeometry
	writer.WriteUInt32At(60, uint32(footer.DiskType))
	writer.WriteBytesAt(68, footer.UniqueID.ToByteSlice())
	writer.WriteBooleanAt(84, footer.SavedState)
	writer.WriteBytesAt(85, footer.Reserved)
	// + Checksum
	//
	// Checksum is one’s complement of the sum of all the bytes in the footer without the
	// checksum field.
	checkSum := uint32(0)
	for i := int(0); i < int(vhdcore.VhdFooterSize); i++ {
		if i < vhdcore.VhdFooterChecksumOffset || i >= vhdcore.VhdFooterChecksumOffset+4 {
			checkSum += uint32(buffer[i])
		}
	}

	writer.WriteUInt32At(64, ^checkSum)
	// - Checksum

	return buffer
}
