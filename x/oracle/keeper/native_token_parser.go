package keeper

import (
	"errors"

	"github.com/imua-xyz/imuachain/x/oracle/types"
)

// TODO: This conversion has limited length for balance change, it suites for beaconchain currently, If we extend to other changes, this method need to be upgrade
// for value that might be too big leading too long length of the change value, many related changes need to be done since the message size might be too big then
// parseBalanceChange parses rawData to details of amount change for all stakers relative to native restaking
func parseBalanceChangeCapped(rawData []byte, sl types.StakerList) (map[string]int, error) {
	// eg. 0100-000011
	// first part 0100 tells that the effective-balance of staker corresponding to index 2 in StakerList
	// the left part 000011. we use the first 4 bits to tell the length of this number, and it shows as 1 here, the 5th bit is used to tell symbol of the number, 1 means negative, then we can get the abs number indicate by the length. It's -1 here, means effective-balane is 32-1 on beacon chain for now
	// the first 32 bytes are information to indicates effective-balance of which staker has changed, 1 means changed, 0 means not. 32 bytes can represents changes for at most 256 stakers
	indexes := rawData[:32]
	// bytes after first 32 are details of effective-balance change for each staker which has been marked with 1 in the first 32 bytes, for those who are marked with 0 will just be ignored
	// For each staker we support at most 256 validators to join, so the biggest effective-balance change we would have is 256*32, then we need 13 bits to represents the number for each staker. And for compression we use 4 bits to tell the length of bits without leading 0 this number has.
	// Then with the symbol we need at most 18 bits for each staker's effective-balance change: 0000.0.0000-0000-0000 (the leading 0 will be ignored for the last 13 bits)
	changes := rawData[32:]
	index := -1
	byteIndex := 0
	bitOffset := 0
	lengthBits := 5
	stakerChanges := make(map[string]int)
	for _, b := range indexes {
		for i := 7; i >= 0; i-- {
			index++
			if (b>>i)&1 == 1 {
				lenValue := changes[byteIndex] << bitOffset
				bitsLeft := 8 - bitOffset
				lenValue >>= (8 - lengthBits)
				if bitsLeft < lengthBits {
					byteIndex++
					lenValue |= changes[byteIndex] >> (8 - lengthBits + bitsLeft)
					bitOffset = lengthBits - bitsLeft
				} else {
					if bitOffset += lengthBits; bitOffset == 8 {
						bitOffset = 0
					}
					if bitsLeft == lengthBits {
						byteIndex++
					}
				}

				symbol := lenValue & 1
				lenValue >>= 1
				if lenValue <= 0 {
					// the range of length we accept is 1-15(the max we will use is actually 13)
					return stakerChanges, errors.New("length of change value must be at least 1 bit")
				}

				bitsExtracted := 0
				stakerChange := 0
				for bitsExtracted < int(lenValue) {
					bitsLeft := 8 - bitOffset
					byteValue := changes[byteIndex] << bitOffset
					if (int(lenValue) - bitsExtracted) < bitsLeft {
						bitsLeft = int(lenValue) - bitsExtracted
						bitOffset += bitsLeft
					} else {
						byteIndex++
						bitOffset = 0
					}
					byteValue >>= (8 - bitsLeft)
					stakerChange = (stakerChange << bitsLeft) | int(byteValue)
					bitsExtracted += bitsLeft
				}
				stakerChange++
				if symbol == 1 {
					stakerChange *= -1
				}
				stakerChanges[sl.StakerAddrs[index]] = stakerChange
			}
		}
	}
	return stakerChanges, nil
}
