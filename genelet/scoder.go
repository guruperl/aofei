package genelet

import (
	"encoding/base64"
)

func Scoder(text string, CRYPTEXT string) []byte {
	len_cryptext := len(CRYPTEXT)
	cryptext := []byte(CRYPTEXT)

	len_text := len(text)
	out := make([]byte, len_text)
	k := len_cryptext/2
	for i, c := range []byte(text) {
		out[i], k = scode_crypt(cryptext, len_cryptext, c, k)
	}

	return out
}

func scode_crypt(cryptext []byte, len_cryptext int, buf byte, i int) (byte, int) {
	//buf ^= 255 & (cryptext[i] ^ (cryptext[0]*byte(i)))
	buf ^= cryptext[i] ^ (cryptext[0]*byte(255&i))
    if (i<(len_cryptext-1)) {
		cryptext[i] += cryptext[i+1]
	} else {
		cryptext[i] += cryptext[0]
	}
	if (cryptext[i]==0) {
		cryptext[i] += 1
	}
	i++
    if i >= len_cryptext {
		i = 0
	}
    return buf, i;
}

func Encode_scoder(text string, CRYPTEXT string) string {
  return base64.StdEncoding.EncodeToString(Scoder(text, CRYPTEXT))
}

func Decode_scoder(text string, CRYPTEXT string) string {
	data, _ := base64.StdEncoding.DecodeString(text)
	return string(Scoder(string(data), CRYPTEXT))
}
