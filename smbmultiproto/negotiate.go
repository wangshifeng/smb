package smbmultiproto

import (
	"github.com/gentlemanautomaton/smb/smbdialect"
	"github.com/gentlemanautomaton/smb/smbtype"
)

// negotiate interprets a slice of bytes as an SMB multi-protocol negotiate
// packet compatible with SMB version 1.
//
// https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-cifs/25c8c3c9-58fc-4bb8-aa8f-0272dede84c5
type negotiate []byte

// Dialect returns any SMB2 dialects present in the negotiate request.
// It returns nil if the request is invalid or does not contain an SMB2
// dialect.
func (n negotiate) Dialects() smbdialect.List {
	if len(n) < 3 {
		return nil
	}

	// Negotiate packets have no parameters. The parameter word count must
	// be zero.
	wordCount := n[0]
	if wordCount != 0 {
		return nil
	}

	// Negotiate packets must have at least two bytes of dialect data
	length := smbtype.Uint16(n[1:3])
	if length < 2 {
		return nil
	}

	// The data must not be truncated
	totalLength := 3 + int(length)
	if len(n) < totalLength {
		return nil
	}

	// Scan the dialects for SMB2
	var (
		hasWildcard = false
		hasSMB202   = false
		data        = []byte(n[3:totalLength])
		cut         = 1
	)
	for i := range data {
		// Validate buffer formats
		if i < cut {
			if format := data[i]; format != 0x02 {
				// Not a null-terminated string
				return nil
			}
			continue
		}

		// Scan for null
		if data[i] != 0 {
			continue
		}

		// Compare dialects
		if i-cut > 0 {
			dialect := data[cut:i]
			if matchesDialect(dialect, smbWildcard) {
				hasWildcard = true
			}
			if matchesDialect(dialect, smb2002) {
				hasSMB202 = true
			}
		}

		// Start the next string after the null and the format byte
		cut = i + 2
	}

	switch {
	case hasWildcard && hasSMB202:
		return dialectBoth
	case hasWildcard:
		return dialectWildcard
	case hasSMB202:
		return dialectSMB202
	default:
		return nil
	}
}

func matchesDialect(value []byte, dialect string) bool {
	if len(value) != len(dialect) {
		return false
	}
	for k := range value {
		if value[k] != dialect[k] {
			return false
		}
	}
	return true
}
