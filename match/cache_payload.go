package match

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

var cachePayloadMagic = []byte{'A', 'O', 'F', 'C'}

const (
	cachePayloadVersionRAdvs    uint8 = 3
	cachePayloadVersionAudience uint8 = 1
	cachePayloadVersionCreative uint8 = 1

	cachePayloadKindRAdvs    uint8 = 1
	cachePayloadKindAudience uint8 = 2
	cachePayloadKindCreative uint8 = 3
)

func packCachePayload(kind, version uint8, body []byte) []byte {
	data := make([]byte, 0, len(cachePayloadMagic)+2+len(body))
	data = append(data, cachePayloadMagic...)
	data = append(data, kind, version)
	data = append(data, body...)
	return data
}

func unpackCachePayload(data []byte, kind, version uint8) ([]byte, error) {
	if len(data) < len(cachePayloadMagic)+2 || !bytes.Equal(data[:len(cachePayloadMagic)], cachePayloadMagic) {
		return data, nil
	}
	gotKind := data[len(cachePayloadMagic)]
	gotVersion := data[len(cachePayloadMagic)+1]
	if gotKind != kind {
		return nil, fmt.Errorf("cache payload kind %d does not match expected kind %d", gotKind, kind)
	}
	if gotVersion != version {
		return nil, fmt.Errorf("unsupported cache payload version %d for kind %d", gotVersion, kind)
	}
	return data[len(cachePayloadMagic)+2:], nil
}

func readAllCachePayload(r io.Reader, kind, version uint8) ([]byte, error) {
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return nil, err
	}
	return unpackCachePayload(buf.Bytes(), kind, version)
}

func writeCachePayloadHeader(w io.Writer, kind, version uint8) error {
	if _, err := w.Write(cachePayloadMagic); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, kind); err != nil {
		return err
	}
	return binary.Write(w, binary.LittleEndian, version)
}
