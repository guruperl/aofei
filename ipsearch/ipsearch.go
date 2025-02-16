// Package ipsearch provides functionality to search IP addresses and retrieve geographical information.
package ipsearch

import (
	"bytes"
	"database/sql"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/prebid/openrtb/v20/adcom1"
)

type IPSearch struct {
	data               []byte
	prefixMap          map[uint32]*prefixIndex
	firstStartIPOffset uint32
	prefixStartOffset  uint32
	prefixEndOffset    uint32
	prefixCount        uint32
}

func LoadIPData(fn string) (*IPSearch, error) {
	//加载ip地址库信息
	data, err := os.ReadFile(fn)
	if err != nil {
		return nil, err
	}

	firstOffset := bytesToLong(data[0], data[1], data[2], data[3])
	preStart := bytesToLong(data[8], data[9], data[10], data[11])
	preEnd := bytesToLong(data[12], data[13], data[14], data[15])
	count := (preEnd-preStart)/9 + 1 // 前缀区块每组

	// 初始化前缀对应索引区区间
	prefixMap := make(map[uint32]*prefixIndex)
	indexBuffer := data[preStart:(preEnd + 9)]
	for k := uint32(0); k < count; k++ {
		i := k * 9
		prefix := uint32(indexBuffer[i] & 0xFF)
		prefixMap[prefix] = getPrefixIndex(indexBuffer, i)

	}
	return &IPSearch{data, prefixMap, firstOffset, preStart, preEnd, count}, nil
}

func (self *IPSearch) CreatePzGeo(ip string) (*PzGeo, error) {
	g, err := self.getIPIndex(ip)
	if err != nil {
		return nil, err
	}
	if g == nil || g.LocalString == nil {
		return nil, fmt.Errorf("ip location not found")
	}
	loc := string(g.LocalString)
	arr := strings.SplitN(loc, "|", 7)
	if len(arr) != 7 {
		return &PzGeo{g.Geo, "", "", "", "", "", "", loc}, nil
	}

	return &PzGeo{
		Geo:       g.Geo,
		Continent: arr[0],
		Country:   arr[1],
		State:     arr[2],
		Metro:     arr[3],
		City:      arr[4],
		Zip:       arr[5],
		Isp:       arr[6],
	}, nil
}

func (self *IPSearch) getIPIndex(ip string) (*ipIndex, error) {
	data := self.data
	leftoffset, localoffset, locallength, err := self.getLocation(ip)
	if err != nil {
		return nil, err
	}
	if leftoffset == 0 {
		return nil, fmt.Errorf("ip location not found")
	}

	geo := new(Geo)
	buf := bytes.NewReader(data[localoffset : localoffset+33])
	if err = binary.Read(buf, binary.LittleEndian, geo); err != nil {
		return nil, err
	}

	return &ipIndex{
		StartIP:     bytesToLong(data[leftoffset], data[1+leftoffset], data[2+leftoffset], data[3+leftoffset]),
		EndIP:       bytesToLong(data[4+leftoffset], data[5+leftoffset], data[6+leftoffset], data[7+leftoffset]),
		LocalOffset: localoffset,
		LocalLength: locallength,
		Geo:         *geo,
		LocalString: data[localoffset+33 : localoffset+locallength],
	}, nil
}

/*
	func (self *IPSearch) Get(ip string) (string, error) {
		leftoffset, localoffset, locallength, err := self.getLocation(ip)
		if err != nil {
			return "", err
		}
		if leftoffset == 0 {
			return "", fmt.Errorf("ip location not found")
		}
		return string(self.data[localoffset+33 : localoffset+locallength]), nil
	}
*/

func (self *IPSearch) GetSimple(ip string) (string, error) {
	leftoffset, localoffset, locallength, err := self.getLocation(ip)
	if err != nil {
		return "", err
	}
	if leftoffset == 0 {
		return "", fmt.Errorf("ip location not found")
	}
	return string(self.data[localoffset : localoffset+locallength]), nil
}

func (self *IPSearch) getLocation(ip string) (uint32, uint32, uint32, error) {
	ips := strings.Split(ip, ".")
	x, err := strconv.ParseUint(ips[0], 10, 32)
	if err != nil {
		return 0, 0, 0, err
	}
	prefix := uint32(x)
	intIP, err := IPToLong(ip)
	if err != nil {
		return 0, 0, 0, err
	}

	var high uint32 = 0
	var low uint32 = 0

	if _, ok := self.prefixMap[prefix]; ok {
		low = self.prefixMap[prefix].StartIndex
		high = self.prefixMap[prefix].EndIndex
	} else {
		return 0, 0, 0, fmt.Errorf("prefix not found")
	}

	left := self.binarySearch(low, high, intIP)
	if left == 0 {
		return 0, 0, 0, fmt.Errorf("binary search failed")
	}

	leftoffset := self.firstStartIPOffset + left*12
	localoffset := bytesToLong3(self.data[8+leftoffset], self.data[9+leftoffset], self.data[10+leftoffset])
	locallength := uint32(self.data[11+leftoffset])

	return leftoffset, localoffset, locallength, nil
}

// 二分逼近算法
func (self *IPSearch) binarySearch(l uint32, h uint32, ip uint32) uint32 {
	if l == h {
		return l
	}
	mid := (l + h) / 2
	for l < h {
		startipNum, endipNum := self.getStartEndIP(mid)
		if ip == startipNum || ip == endipNum || (ip > startipNum && ip < endipNum) {
			return mid
		} else if ip < startipNum {
			h = mid
		} else if ip > endipNum {
			l = mid
		}

		if (h - l) == 1 {
			startipNum, endipNum = self.getStartEndIP(l)
			if ip == startipNum || ip == endipNum || (ip > startipNum && ip < endipNum) {
				return l
			}
			startipNum, endipNum = self.getStartEndIP(h)
			if ip == startipNum || ip == endipNum || (ip > startipNum && ip < endipNum) {
				return h
			}
			return 0
		}
		mid = (l + h) / 2
	}
	return mid
}

// 只获取结束ip的数值
// 索引区第left个索引
// 返回结束ip的数值
func (self *IPSearch) getStartEndIP(left uint32) (uint32, uint32) {
	leftoffset := self.firstStartIPOffset + left*12
	return bytesToLong(self.data[0+leftoffset], self.data[1+leftoffset], self.data[2+leftoffset], self.data[3+leftoffset]), bytesToLong(self.data[4+leftoffset], self.data[5+leftoffset], self.data[6+leftoffset], self.data[7+leftoffset])
}

func DatabaseToDat(db *sql.DB, outfile string) error {
	_, err := db.Exec("SET NAMES 'utf8mb4'")
	if err != nil {
		return err
	}
	rows, err := db.Query(`SELECT ip_start, ip_end, ip_start_num, ip_end_num, IFNULL(CONCAT(continent, "|", country, "|", province, "|", city, "|", district, "|", area_code, "|", isp),"") AS geo, IFNULL(continent_id,0), IFNULL(country_id,0), IFNULL(state_id,0), IFNULL(dma_id,0), IFNULL(city_id, 0), IFNULL(isp_id,0), IFNULL(area_code,"") AS zip, IFNULL(latitude,""), IFNULL(longitude,"") FROM ip`)
	if err != nil {
		return err
	}
	defer rows.Close()

	total := 16
	n := 0
	ind := make([]*ipIndex, 0)
	keys := make([]uint32, 0)
	pre := make(map[uint32]*prefixIndex)

	for rows.Next() {
		var ipStart, ipEnd string
		geo := make([]byte, 0)
		var ipStartnum, ipEndnum uint32
		var continentID uint8
		var countryID, stateID, dmaID, ispID uint16
		var cityID uint32
		var zip, latitude, longitude string
		if err := rows.Scan(&ipStart, &ipEnd, &ipStartnum, &ipEndnum, &geo, &continentID, &countryID, &stateID, &dmaID, &cityID, &ispID, &zip, &latitude, &longitude); err != nil {
			return err
		}

		ll := len(latitude)
		if ll > 0 && latitude[ll-1] == '\x0d' {
			latitude = latitude[:ll-1]
		}

		clean := make([]byte, 0)
		for _, c := range geo {
			if c != '\x0d' {
				clean = append(clean, c)
			}
		}
		length := len(clean)

		zip32, err := strconv.ParseUint(zip, 10, 32)
		if err != nil {
			zip32 = 0
		}
		lat64, err := strconv.ParseFloat(latitude, 64)
		if err != nil {
			lat64 = 0.0
		}
		lon64, err := strconv.ParseFloat(longitude, 64)
		if err != nil {
			lon64 = 0.0
		}

		ids := Geo{
			ContinentID: continentID,
			CountryID:   countryID,
			StateID:     stateID,
			DmaID:       dmaID,
			CityID:      cityID,
			IspID:       ispID,
			ZipID:       uint32(zip32),
			Location: Location{
				Lat:       lat64,
				Lon:       lon64,
				Type:      adcom1.LocationIP,
				Accuracy:  0,
				LastFix:   0,
				IPService: adcom1.LocationServiceMaxMind,
			},
		}
		length += 33 // 1+2+2+2+4+2
		if length >= 255 {
			panic(err)
		}
		ind = append(ind, &ipIndex{ipStartnum, ipEndnum, uint32(total), uint32(length), ids, clean})

		ips := strings.Split(ipStart, ".")
		x, _ := strconv.ParseUint(ips[0], 10, 32)
		prefix := uint32(x)

		if _, ok := pre[prefix]; ok {
			pre[prefix].EndIndex = uint32(n)
		} else {
			keys = append(keys, prefix)
			pre[prefix] = &prefixIndex{uint32(n), uint32(n)}
		}

		n++
		total += length
	}
	if err := rows.Err(); err != nil {
		return err
	}

	a0 := uint32(total)
	total += 12 * n
	a1 := uint32(total - 12)
	a2 := uint32(total)
	total += 9 * len(keys)
	a3 := uint32(total - 9)

	out, err := os.Create(outfile)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = out.Write(get4(a0))
	if err != nil {
		return err
	}
	_, err = out.Write(get4(a1))
	if err != nil {
		return err
	}
	_, err = out.Write(get4(a2))
	if err != nil {
		return err
	}
	_, err = out.Write(get4(a3))
	if err != nil {
		return err
	}

	for i := 0; i < n; i++ {
		buf := new(bytes.Buffer)
		err = binary.Write(buf, binary.LittleEndian, &ind[i].Geo)
		if err != nil {
			return err
		}
		_, err = out.Write(buf.Bytes())
		if err != nil {
			return err
		}
		_, err = out.Write(ind[i].LocalString)
		if err != nil {
			return err
		}
	}

	for i := 0; i < n; i++ {
		_, err = out.Write(get4(ind[i].StartIP))
		if err != nil {
			return err
		}
		_, err = out.Write(get4(ind[i].EndIP))
		if err != nil {
			return err
		}
		_, err = out.Write(get3(ind[i].LocalOffset))
		if err != nil {
			return err
		}
		_, err = out.Write([]byte{byte(ind[i].LocalLength)})
		if err != nil {
			return err
		}
	}

	for _, k := range keys {
		_, err = out.Write([]byte{byte(k)})
		if err != nil {
			return err
		}
		_, err = out.Write(get4(pre[k].StartIndex))
		if err != nil {
			return err
		}
		_, err = out.Write(get4(pre[k].EndIndex))
		if err != nil {
			return err
		}
	}

	return nil
}
