package grabtest

import "encoding/hex"

func MustHexDecodeString(s string) (b []byte) {
	var err error
	b, err = hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return
}

func MustHexEncodeString(b []byte) (s string) {
	return hex.EncodeToString(b)
}
