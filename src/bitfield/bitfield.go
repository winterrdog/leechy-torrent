package bitfield

// Bitfield represents a bitfield used in a BitTorrent client.
type Bitfield []byte

// HasPiece checks if the specified piece index is present in the Bitfield.
// It returns true if the piece is present, otherwise false.
func (bf *Bitfield) HasPiece(index int) bool {
	var byteIndex = index / 8 // byte to consider
	var offset = index % 8    // position within a byte

	if byteIndex < 0 || byteIndex >= len(*bf) {
		return false
	}

	var bitValue = (*bf)[byteIndex] >> (7 - offset)
	return bitValue&1 == 1
}

// SetPiece sets the bit at the specified index in the Bitfield.
// It calculates the byte index and offset within the byte based on the given index.
// If the index is out of bounds, it silently ignores the operation.
// The bit at the specified index is set to 1 using bitwise OR operation.
func (bf *Bitfield) SetPiece(index int) {
	var byteIndex = index / 8 // byte to consider
	var offset = index % 8    // position within a byte

	// silently ignore if index is out of bounds
	if byteIndex < 0 || byteIndex >= len(*bf) {
		return
	}

	(*bf)[byteIndex] |= 1 << (7 - offset)
}
