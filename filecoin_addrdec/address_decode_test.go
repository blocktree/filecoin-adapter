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

	check := dec.AddressVerify(addr)
	t.Logf("check: %v \n", check)
}


func TestAddressDecoder_VerifyAddress(t *testing.T) {
	dec := NewAddressDecoderV2(true)
	check := dec.AddressVerify("t1ojyfm5btrqq63zquewexr4hecynvq6yjyk5xv6q")
	t.Logf("check: %v \n", check)
}
