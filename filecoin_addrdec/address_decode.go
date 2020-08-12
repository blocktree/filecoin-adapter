package filecoin_addrdec

import (
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"github.com/blocktree/go-owaddress"
	"github.com/blocktree/go-owcrypt"
	"github.com/blocktree/openwallet/v2/openwallet"
	"github.com/minio/blake2b-simd"
	"strings"
)

const(
	MainnetPrefix = "f" // MainnetPrefix is the main network prefix.
	TestnetPrefix = "t" // TestnetPrefix is the main network prefix.
	PayloadHashLength = 20 // PayloadHashLength defines the hash length taken over addresses using the Actor and SECP256K1 protocols.
	ChecksumHashLength = 4 // ChecksumHashLength defines the hash length used for calculating address checksums.
	EncodeStd = "abcdefghijklmnopqrstuvwxyz234567"
)

var (
	payloadHashConfig = &blake2b.Config{Size: PayloadHashLength}
	checksumHashConfig = &blake2b.Config{Size: ChecksumHashLength}
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
func (dec *AddressDecoderV2) AddressEncode(hash []byte, opts ...interface{}) (string, error) {
	if len(hash) != 32 {
		//公钥hash处理
		hash = owcrypt.PointDecompress(hash, owcrypt.ECC_CURVE_SECP256K1)
		//hash = owcrypt.Hash(publicKey[1:len(publicKey)], 0, owcrypt.HASH_ALG_KECCAK256)
	}else{
		ret := make([]byte, 65)
		ret[0] = 4 // uncompressed point
		copy(ret[1:65], hash)
	}

	//fmt.Println("decompress pub : ", hex.EncodeToString(hash), ", publickey length: ", len(hash) )

	hash = addressHash(hash, payloadHashConfig)

	protocol := byte(0x01)

	explen := 1 + len(hash)
	buf := make([]byte, explen)

	buf[0] = protocol
	copy(buf[1:], hash)

	//return Address{string(buf)}, nil
	str := string(buf)

	ntwk := MainnetPrefix
	if dec.IsTestNet {
		ntwk = TestnetPrefix
	}

	var AddressEncoding = base32.NewEncoding(EncodeStd)
	payLoad := []byte(str[1:])

	waitCheckSum := append([]byte{protocol}, payLoad...)
	cksm := Checksum(waitCheckSum)
	address := ntwk + fmt.Sprintf("%d", protocol) + AddressEncoding.WithPadding(-1).EncodeToString(append(payLoad, cksm[:]...))

	return address, nil
}

func addressHash(ingest []byte, cfg *blake2b.Config) []byte {
	hasher, err := blake2b.New(cfg)
	if err != nil {
		// If this happens sth is very wrong.
		panic(fmt.Sprintf("invalid address hash configuration: %v", err)) // ok
	}
	if _, err := hasher.Write(ingest); err != nil {
		// blake2bs Write implementation never returns an error in its current
		// setup. So if this happens sth went very wrong.
		panic(fmt.Sprintf("blake2b is unable to process hashes: %v", err)) // ok
	}
	return hasher.Sum(nil)
}

// Checksum returns the checksum of `ingest`.
func Checksum(ingest []byte) []byte {
	return addressHash(ingest, checksumHashConfig)
}

// AddressVerify 地址校验
func (dec *AddressDecoderV2) AddressVerify(address string, opts ...interface{}) bool {
	valid, err := owaddress.Verify("eth", address)
	if err != nil {
		return false
	}
	return valid
}
