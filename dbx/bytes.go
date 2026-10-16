package dbx

// NonNil converts nil to an empty slice for a NOT NULL blob column, which
// would otherwise receive NULL.
func NonNil(b []byte) []byte {
	if b == nil {
		return []byte{}
	}

	return b
}
