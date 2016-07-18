// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build 386 arm arm64 ppc64 ppc64le appengine

package crc32

// The file contains the generic version of updateCastagnoli which does
// slicing-by-8, or uses the fallback for very small sizes.
func updateCastagnoli(crc uint32, p []byte) uint32 {
	// only use slicing-by-8 when input is >= 16 Bytes
	if len(p) >= 16 {
		return updateSlicingBy8(crc, castagnoliTable8, p)
	}
	return update(crc, castagnoliTable, p)
}

func updateIEEE(crc uint32, p []byte) uint32 {
	// only use slicing-by-8 when input is >= 16 Bytes
	if len(p) >= 16 {
		iEEETable8Once.Do(func() {
			iEEETable8 = makeTable8(IEEE)
		})
		return updateSlicingBy8(crc, iEEETable8, p)
	}
	return update(crc, IEEETable, p)
}
