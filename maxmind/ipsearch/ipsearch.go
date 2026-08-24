package ipsearch

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"strconv"
	"strings"

	"github.com/guruperl/aofei/internal/atomicfile"
)

const (
	datHeaderSize   = uint32(16)
	datGeoSize      = uint32(33)
	datIndexSize    = uint32(12)
	datPrefixSize   = uint32(9)
	maxDataOffset   = uint32(1<<24 - 1)
	maxRecordLength = uint32(254)
)

type IPSearch struct {
	data               []byte
	prefixMap          map[uint32]*prefixIndex
	firstStartIPOffset uint32
	prefixStartOffset  uint32
	prefixEndOffset    uint32
	prefixCount        uint32
	indexCount         uint32
}

func LoadIPData(fn string) (*IPSearch, error) {
	data, err := os.ReadFile(fn)
	if err != nil {
		return nil, err
	}
	search, err := parseIPData(data)
	if err != nil {
		return nil, fmt.Errorf("load legacy IP data %q: %w", fn, err)
	}
	return search, nil
}

func parseIPData(data []byte) (*IPSearch, error) {
	if uint64(len(data)) < uint64(datHeaderSize) {
		return nil, fmt.Errorf("header is truncated: got %d bytes, need %d", len(data), datHeaderSize)
	}
	firstOffset := binary.LittleEndian.Uint32(data[0:4])
	indexEnd := binary.LittleEndian.Uint32(data[4:8])
	preStart := binary.LittleEndian.Uint32(data[8:12])
	preEnd := binary.LittleEndian.Uint32(data[12:16])
	dataLength := uint64(len(data))

	if firstOffset < datHeaderSize || uint64(firstOffset) > dataLength {
		return nil, fmt.Errorf("first index offset %d is outside [%d,%d]", firstOffset, datHeaderSize, len(data))
	}
	if preStart < firstOffset || uint64(preStart) > dataLength {
		return nil, fmt.Errorf("prefix start offset %d precedes the index or exceeds the file", preStart)
	}
	indexBytes := preStart - firstOffset
	if indexBytes == 0 || indexBytes%datIndexSize != 0 {
		return nil, fmt.Errorf("index region length %d is not a non-empty multiple of %d", indexBytes, datIndexSize)
	}
	indexCount := indexBytes / datIndexSize
	wantIndexEnd := uint64(firstOffset) + uint64(indexCount-1)*uint64(datIndexSize)
	if uint64(indexEnd) != wantIndexEnd {
		return nil, fmt.Errorf("index end offset %d does not match %d entries ending at %d", indexEnd, indexCount, wantIndexEnd)
	}
	if preEnd < preStart || (preEnd-preStart)%datPrefixSize != 0 {
		return nil, fmt.Errorf("prefix end offset %d does not align with prefix start %d", preEnd, preStart)
	}
	prefixEndExclusive := uint64(preEnd) + uint64(datPrefixSize)
	if prefixEndExclusive != dataLength {
		return nil, fmt.Errorf("prefix region ends at %d, file length is %d", prefixEndExclusive, len(data))
	}
	prefixCount := (preEnd-preStart)/datPrefixSize + 1

	search := &IPSearch{
		data:               data,
		prefixMap:          make(map[uint32]*prefixIndex, prefixCount),
		firstStartIPOffset: firstOffset,
		prefixStartOffset:  preStart,
		prefixEndOffset:    preEnd,
		prefixCount:        prefixCount,
		indexCount:         indexCount,
	}
	if err := search.validateIndexes(); err != nil {
		return nil, err
	}
	if err := search.loadPrefixes(); err != nil {
		return nil, err
	}
	return search, nil
}

func (self *IPSearch) validateIndexes() error {
	var priorEnd uint32
	for index := uint32(0); index < self.indexCount; index++ {
		record, err := self.indexRecord(index)
		if err != nil {
			return err
		}
		startIP := binary.LittleEndian.Uint32(record[0:4])
		endIP := binary.LittleEndian.Uint32(record[4:8])
		if startIP > endIP {
			return fmt.Errorf("index %d has descending IP range %d-%d", index, startIP, endIP)
		}
		if index != 0 && startIP <= priorEnd {
			return fmt.Errorf("index %d overlaps or is out of order", index)
		}
		priorEnd = endIP

		localOffset := uint32(record[8]) | uint32(record[9])<<8 | uint32(record[10])<<16
		localLength := uint32(record[11])
		if localLength < datGeoSize {
			return fmt.Errorf("index %d location length %d is shorter than geo record %d", index, localLength, datGeoSize)
		}
		if localOffset < datHeaderSize || uint64(localOffset)+uint64(localLength) > uint64(self.firstStartIPOffset) {
			return fmt.Errorf("index %d location range [%d,%d) is outside the data region [%d,%d)", index, localOffset, uint64(localOffset)+uint64(localLength), datHeaderSize, self.firstStartIPOffset)
		}
	}
	return nil
}

func (self *IPSearch) loadPrefixes() error {
	for index := uint32(0); index < self.prefixCount; index++ {
		offset := uint64(self.prefixStartOffset) + uint64(index)*uint64(datPrefixSize)
		record, err := boundedSlice(self.data, offset, uint64(datPrefixSize))
		if err != nil {
			return fmt.Errorf("prefix index %d: %w", index, err)
		}
		prefix := uint32(record[0])
		if _, duplicate := self.prefixMap[prefix]; duplicate {
			return fmt.Errorf("prefix %d is duplicated", prefix)
		}
		bounds := GetPrefixIndex(record, 0)
		if bounds == nil || bounds.StartIndex > bounds.EndIndex || bounds.EndIndex >= self.indexCount {
			return fmt.Errorf("prefix %d has invalid index range", prefix)
		}
		self.prefixMap[prefix] = bounds
	}
	return nil
}

func (self *IPSearch) CreatePzGeo(ip string) (*PzGeo, error) {
	g, err := self.getIPIndex(ip)
	if err != nil {
		return nil, err
	}
	if g == nil || g.LocalString == nil {
		return new(PzGeo), nil
	}
	loc := string(g.LocalString)
	arr := strings.SplitN(loc, "|", 7)
	if len(arr) != 7 {
		return &PzGeo{g.Geo, "", "", "", "", "", "", loc}, nil
	}

	return &PzGeo{g.Geo, arr[0], arr[1], arr[2], arr[3], arr[4], arr[5], arr[6]}, nil
}

func (self *IPSearch) getIPIndex(ip string) (*ipIndex, error) {
	leftOffset, localOffset, localLength, err := self.getLocation(ip)
	if err != nil || leftOffset == 0 {
		return nil, err
	}
	record, err := boundedSlice(self.data, uint64(localOffset), uint64(localLength))
	if err != nil {
		return nil, fmt.Errorf("location record: %w", err)
	}
	geo := new(Geo)
	if err := binary.Read(bytes.NewReader(record[:datGeoSize]), binary.LittleEndian, geo); err != nil {
		return nil, err
	}
	index, err := boundedSlice(self.data, uint64(leftOffset), uint64(datIndexSize))
	if err != nil {
		return nil, fmt.Errorf("IP index: %w", err)
	}
	return &ipIndex{
		StartIP:     binary.LittleEndian.Uint32(index[0:4]),
		EndIP:       binary.LittleEndian.Uint32(index[4:8]),
		LocalOffset: localOffset,
		LocalLength: localLength,
		Geo:         *geo,
		LocalString: record[datGeoSize:],
	}, nil
}

func (self *IPSearch) Get(ip string) (string, error) {
	leftOffset, localOffset, localLength, err := self.getLocation(ip)
	if err != nil || leftOffset == 0 {
		return "", err
	}
	record, err := boundedSlice(self.data, uint64(localOffset), uint64(localLength))
	if err != nil {
		return "", err
	}
	return string(record[datGeoSize:]), nil
}

func (self *IPSearch) GetSimple(ip string) (string, error) {
	leftOffset, localOffset, localLength, err := self.getLocation(ip)
	if err != nil || leftOffset == 0 {
		return "", err
	}
	record, err := boundedSlice(self.data, uint64(localOffset), uint64(localLength))
	if err != nil {
		return "", err
	}
	return string(record), nil
}

func (self *IPSearch) getLocation(ip string) (uint32, uint32, uint32, error) {
	addr, err := netip.ParseAddr(ip)
	if err != nil || !addr.Is4() {
		return 0, 0, 0, fmt.Errorf("invalid IPv4 address %q", ip)
	}
	quads := addr.As4()
	prefix := uint32(quads[0])
	intIP := binary.BigEndian.Uint32(quads[:])
	bounds, ok := self.prefixMap[prefix]
	if !ok {
		return 0, 0, 0, nil
	}

	left, found, err := self.binarySearch(bounds.StartIndex, bounds.EndIndex, intIP)
	if err != nil || !found {
		return 0, 0, 0, err
	}
	record, err := self.indexRecord(left)
	if err != nil {
		return 0, 0, 0, err
	}
	leftOffset := self.firstStartIPOffset + left*datIndexSize
	localOffset := uint32(record[8]) | uint32(record[9])<<8 | uint32(record[10])<<16
	localLength := uint32(record[11])
	if _, err := boundedSlice(self.data, uint64(localOffset), uint64(localLength)); err != nil {
		return 0, 0, 0, fmt.Errorf("index %d location: %w", left, err)
	}
	return leftOffset, localOffset, localLength, nil
}

func (self *IPSearch) binarySearch(low, high, ip uint32) (uint32, bool, error) {
	if low > high || high >= self.indexCount {
		return 0, false, fmt.Errorf("search range [%d,%d] exceeds %d indexes", low, high, self.indexCount)
	}
	for low <= high {
		mid := low + (high-low)/2
		startIP, endIP, err := self.getStartEndIP(mid)
		if err != nil {
			return 0, false, err
		}
		switch {
		case ip < startIP:
			if mid == 0 {
				return 0, false, nil
			}
			high = mid - 1
		case ip > endIP:
			low = mid + 1
		default:
			return mid, true, nil
		}
	}
	return 0, false, nil
}

func (self *IPSearch) getStartEndIP(index uint32) (uint32, uint32, error) {
	record, err := self.indexRecord(index)
	if err != nil {
		return 0, 0, err
	}
	return binary.LittleEndian.Uint32(record[0:4]), binary.LittleEndian.Uint32(record[4:8]), nil
}

func (self *IPSearch) indexRecord(index uint32) ([]byte, error) {
	if index >= self.indexCount {
		return nil, fmt.Errorf("index %d exceeds count %d", index, self.indexCount)
	}
	offset := uint64(self.firstStartIPOffset) + uint64(index)*uint64(datIndexSize)
	record, err := boundedSlice(self.data, offset, uint64(datIndexSize))
	if err != nil {
		return nil, fmt.Errorf("index %d: %w", index, err)
	}
	return record, nil
}

func boundedSlice(data []byte, offset, length uint64) ([]byte, error) {
	if offset > uint64(len(data)) || length > uint64(len(data))-offset {
		return nil, fmt.Errorf("range [%d,%d) exceeds %d bytes", offset, offset+length, len(data))
	}
	return data[offset : offset+length], nil
}

func DatabaseToDat(db *sql.DB, outfile string) error {
	if db == nil {
		return errors.New("IP database is nil")
	}
	if _, err := db.Exec("SET NAMES 'utf8mb4'"); err != nil {
		return err
	}
	rows, err := db.Query(`SELECT ip_start, ip_end, ip_start_num, ip_end_num, IFNULL(CONCAT(continent, "|", country, "|", province, "|", city, "|", district, "|", area_code, "|", isp),"") AS geo, IFNULL(continent_id,0), IFNULL(country_id,0), IFNULL(state_id,0), IFNULL(dma_id,0), IFNULL(city_id, 0), IFNULL(isp_id,0), IFNULL(area_code,"") AS zip, IFNULL(latitude,""), IFNULL(longitude,"") FROM ip ORDER BY ip_start_num`)
	if err != nil {
		return err
	}
	defer rows.Close()

	total := uint64(datHeaderSize)
	ind := make([]*ipIndex, 0)
	keys := make([]uint32, 0)
	pre := make(map[uint32]*prefixIndex)
	var priorEnd uint32

	for rows.Next() {
		var ipStart, ipEnd string
		geo := make([]byte, 0)
		var ipStartNum, ipEndNum uint32
		var continentID uint8
		var countryID, stateID, dmaID, ispID uint16
		var cityID uint32
		var zip, latitude, longitude string
		if err := rows.Scan(&ipStart, &ipEnd, &ipStartNum, &ipEndNum, &geo, &continentID, &countryID, &stateID, &dmaID, &cityID, &ispID, &zip, &latitude, &longitude); err != nil {
			return err
		}
		parsedStart, err := IPToLong(ipStart)
		if err != nil || parsedStart != ipStartNum {
			return fmt.Errorf("invalid start IP row %q", ipStart)
		}
		parsedEnd, err := IPToLong(ipEnd)
		if err != nil || parsedEnd != ipEndNum || ipStartNum > ipEndNum {
			return fmt.Errorf("invalid end IP row %q", ipEnd)
		}
		if len(ind) != 0 && ipStartNum <= priorEnd {
			return fmt.Errorf("IP row %q overlaps or is out of order", ipStart)
		}
		priorEnd = ipEndNum

		latitude = strings.TrimSuffix(latitude, "\r")
		clean := bytes.ReplaceAll(geo, []byte{'\r'}, nil)
		length := uint32(len(clean)) + datGeoSize
		if length > maxRecordLength {
			return fmt.Errorf("ip location record for %s is too long: %d bytes", ipStart, length)
		}
		if total > uint64(maxDataOffset) {
			return errors.New("legacy IP data region exceeds 24-bit offsets")
		}

		zip32, _ := strconv.ParseUint(zip, 10, 32)
		lat64, _ := strconv.ParseFloat(latitude, 64)
		lon64, _ := strconv.ParseFloat(longitude, 64)
		ids := Geo{continentID, countryID, stateID, dmaID, cityID, ispID, uint32(zip32), lat64, lon64}
		ind = append(ind, &ipIndex{ipStartNum, ipEndNum, uint32(total), length, ids, clean})

		index := uint32(len(ind) - 1)
		addPrefixRange(&keys, pre, ipStartNum, ipEndNum, index)
		total += uint64(length)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(ind) == 0 {
		return errors.New("legacy IP database contains no rows")
	}
	if total+uint64(len(ind))*uint64(datIndexSize)+uint64(len(keys))*uint64(datPrefixSize) > uint64(^uint32(0)) {
		return errors.New("legacy IP database output exceeds 32-bit offsets")
	}

	return atomicfile.Write(outfile, 0640, func(out io.Writer) error {
		return writeDat(out, ind, keys, pre, uint32(total))
	})
}

func addPrefixRange(keys *[]uint32, prefixes map[uint32]*prefixIndex, startIP, endIP, index uint32) {
	startPrefix := startIP >> 24
	endPrefix := endIP >> 24
	for prefix := startPrefix; prefix <= endPrefix; prefix++ {
		if existing, ok := prefixes[prefix]; ok {
			existing.EndIndex = index
		} else {
			*keys = append(*keys, prefix)
			prefixes[prefix] = &prefixIndex{StartIndex: index, EndIndex: index}
		}
	}
}

func writeDat(out io.Writer, indexes []*ipIndex, keys []uint32, prefixes map[uint32]*prefixIndex, firstIndexOffset uint32) error {
	indexEnd := firstIndexOffset + uint32(len(indexes)-1)*datIndexSize
	prefixStart := firstIndexOffset + uint32(len(indexes))*datIndexSize
	prefixEnd := prefixStart + uint32(len(keys)-1)*datPrefixSize
	for _, value := range []uint32{firstIndexOffset, indexEnd, prefixStart, prefixEnd} {
		if _, err := out.Write(get4(value)); err != nil {
			return err
		}
	}
	for _, index := range indexes {
		if err := binary.Write(out, binary.LittleEndian, &index.Geo); err != nil {
			return err
		}
		if _, err := out.Write(index.LocalString); err != nil {
			return err
		}
	}
	for _, index := range indexes {
		for _, data := range [][]byte{get4(index.StartIP), get4(index.EndIP), get3(index.LocalOffset), {byte(index.LocalLength)}} {
			if _, err := out.Write(data); err != nil {
				return err
			}
		}
	}
	for _, key := range keys {
		prefix := prefixes[key]
		for _, data := range [][]byte{{byte(key)}, get4(prefix.StartIndex), get4(prefix.EndIndex)} {
			if _, err := out.Write(data); err != nil {
				return err
			}
		}
	}
	return nil
}
