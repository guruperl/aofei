package ipsearch

type prefixIndex struct {
	StartIndex uint32
	EndIndex   uint32
}

func GetPrefixIndex(indexBuffer []byte, i uint32) *prefixIndex {
	startIndex := bytesToLong(indexBuffer[i+1], indexBuffer[i+2], indexBuffer[i+3], indexBuffer[i+4])
	endIndex := bytesToLong(indexBuffer[i+5], indexBuffer[i+6], indexBuffer[i+7], indexBuffer[i+8])
	return &prefixIndex{startIndex, endIndex}
}
