package ipsearch

type prefixIndex struct {
	StartIndex uint32
	EndIndex   uint32
}

func getPrefixIndex(indexBuffer []byte, i uint32) *prefixIndex {
	return &prefixIndex{
		StartIndex: bytesToLong(indexBuffer[i+1], indexBuffer[i+2], indexBuffer[i+3], indexBuffer[i+4]),
		EndIndex:   bytesToLong(indexBuffer[i+5], indexBuffer[i+6], indexBuffer[i+7], indexBuffer[i+8]),
	}
}
