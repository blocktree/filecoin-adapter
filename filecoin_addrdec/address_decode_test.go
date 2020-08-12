package filecoin_addrdec

import (
	"encoding/hex"
	"testing"
)

func TestAddressDecoder_AddressEncode(t *testing.T) {
	pub, _ := hex.DecodeString("0265ff85a638b555ad5f15359ef0d80688452bd4dae3a29ecdf53e74b76862a6f2")
	dec := NewAddressDecoderV2(true)
	addr, _ := dec.AddressEncode(pub, false)
	t.Logf("addr: %s", addr)
	//	t1mmls5uxmgab27jyybbm5lxxqmapptvlbo7pes7i
}

func TestAddressDecoder_AddressDecode(t *testing.T) {
	addr := "t1mmls5uxmgab27jyybbm5lxxqmapptvlbo7pes7i"

	dec := NewAddressDecoderV2(true)
	hash, _ := dec.AddressDecode(addr)
	t.Logf("hash: %s", hex.EncodeToString(hash))
}
