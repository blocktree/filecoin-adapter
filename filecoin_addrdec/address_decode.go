package filecoin_addrdec

import (
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"github.com/blocktree/go-owcrypt"
	"github.com/blocktree/openwallet/v2/openwallet"
	"github.com/minio/blake2b-simd"
	"strings"
)

const(
	MainnetPrefix = "f" // MainnetPrefix is the main network prefix.
	TestnetPrefix = "t" // TestnetPrefix is the main network prefix.
	PayloadHashLength = 20 // PayloadHashLength defines the hash length taken over addresses using the Actor and SECP256K1 protocols.
	BlsPublicKeyBytes = 48 // BlsPublicKeyBytes is the length of a BLS public key
	ChecksumHashLength = 4 // ChecksumHashLength defines the hash length used for calculating address checksums.
	EncodeStd = "abcdefghijklmnopqrstuvwxyz234567"
	Secp256k1_Protocol = byte(0x01)
	Bls_Protocol = byte(0x03)
)

var (
	payloadHashConfig = &blake2b.Config{Size: PayloadHashLength}
	checksumHashConfig = &blake2b.Config{Size: ChecksumHashLength}
	addressEncoding = base32.NewEncoding(EncodeStd)
)

//AddressDecoderV2
type AddressDecoderV2 struct {
	*openwallet.AddressDecoderV2Base
	IsTestNet bool
}

//NewAddressDecoder 地址解析器
func NewAddressDecoderV2(isTestNet bool) *AddressDecoderV2 {
	decoder := AddressDecoderV2{}
	decoder.IsTestNet = isTestNet
	return &decoder
}

//AddressDecode 地址解析
func (dec *AddressDecoderV2) AddressDecode(addr string, opts ...interface{}) ([]byte, error) {
	addr = strings.TrimPrefix(addr, "0x")
	decodeAddr, err := hex.DecodeString(addr)
	if err != nil {
		return nil, err
	}
	return decodeAddr, err

}

//AddressEncode 地址编码
func (dec *AddressDecoderV2) AddressEncode(publicKey []byte, opts ...interface{}) (string, error) {
	if len(publicKey) != 32 {
		//公钥hash处理
		publicKey = owcrypt.PointDecompress(publicKey, owcrypt.ECC_CURVE_SECP256K1)
	}

	hash := owcrypt.Hash(publicKey, PayloadHashLength, owcrypt.HASH_ALG_BLAKE2B)
	waitCheckSum := append([]byte{Secp256k1_Protocol}, hash...)
	cksm := owcrypt.Hash(waitCheckSum, ChecksumHashLength, owcrypt.HASH_ALG_BLAKE2B)
	address := dec.GetNtwk() + fmt.Sprintf("%d", Secp256k1_Protocol) + addressEncoding.WithPadding(-1).EncodeToString(append(hash, cksm[:]...))

	return address, nil
}

// AddressVerify 地址校验
func (dec *AddressDecoderV2) AddressVerify(address string, opts ...interface{}) bool {

	if len(address)<41 {
		return false
	}

	prefix := address[:2]

	if prefix != dec.GetNtwk()+fmt.Sprintf("%d", Secp256k1_Protocol) && prefix != dec.GetNtwk()+fmt.Sprintf("%d", Bls_Protocol)  {
		return false
	}

	protocol := Secp256k1_Protocol
	payLoadHashLength := PayloadHashLength
	if prefix == dec.GetNtwk()+fmt.Sprintf("%d", Bls_Protocol) {
		protocol = Bls_Protocol
		payLoadHashLength = BlsPublicKeyBytes
	}

	addressEnd, _ := addressEncoding.WithPadding(-1).DecodeString( address[2:] )

	decodeBytes := append([]byte{protocol}, addressEnd[:payLoadHashLength]...)
	check := owcrypt.Hash(decodeBytes, ChecksumHashLength, owcrypt.HASH_ALG_BLAKE2B)

	if len(addressEnd)<24 || len(check)<4 {
		return false
	}

	for i := 0; i < 4; i++ {
		if check[i] != addressEnd[payLoadHashLength+i] {
			return false
		}
	}
	return true
}

func (dec *AddressDecoderV2) GetNtwk() string {
	ntwk := MainnetPrefix
	if dec.IsTestNet {
		ntwk = TestnetPrefix
	}
	return ntwk
}
