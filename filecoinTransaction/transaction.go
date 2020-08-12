package filecoinTransaction

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blocktree/go-owcrypt"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-crypto"
	"github.com/minio/blake2b-simd"
	"github.com/shopspring/decimal"
)

func (m Message) CreateEmptyTransactionAndMessage() (string, string, error) {

	js, _ := json.Marshal(m)

	cid := m.Cid()
	bytes := cid.Bytes()

	b2sum := blake2b.Sum256(bytes)

	return string(js), hex.EncodeToString(b2sum[:]), nil
}

func NewMessageFromJSON(j string) (*Message, error) {

	m := Message{}

	err := json.Unmarshal([]byte(j), &m)

	if err != nil {
		return nil, err
	}

	return &m, nil
}

func SignTransaction(msgStr string, prikey []byte) ([]byte, error) {
	fmt.Println("msg :", msgStr, ", prikey : ", hex.EncodeToString(prikey) )
	msg, err := hex.DecodeString(msgStr)
	if err != nil || len(msg) == 0 {
		return nil, errors.New("invalid message to sign")
	}

	if prikey == nil || len(prikey) != 32 {
		return nil, errors.New("invalid private key")
	}

	//b2sum := blake2b.Sum256(msg)
	signature, v, retCode := owcrypt.Signature(prikey, nil, msg, owcrypt.ECC_CURVE_SECP256K1)
	if retCode != owcrypt.SUCCESS {
		return nil, errors.New("sign failed")
	}
	signature = append(signature, v)

	return signature, nil
}

func VerifyAndCombineTransaction(emptyTrans, signature string) (string, bool) {
	message, err := NewMessageFromJSON(emptyTrans)
	if err != nil {
		return "", false
	}

	//js, _ := json.Marshal(message)
	//fmt.Println("js : ", string(js))

	cid := message.Cid()
	bytes := cid.Bytes()

	sig, _ := hex.DecodeString( signature )

	b2sum := blake2b.Sum256(bytes)
	pubk, err := crypto.EcRecover(b2sum[:], sig)
	if err != nil {
		return "", false
	}

	maybeaddr, err := address.NewSecp256k1Address(pubk)
	if err != nil {
		return "", false
	}
	if message.From != maybeaddr {
		fmt.Println("maybe : ", maybeaddr.String(), ", from : ", message.From.String() )
		return "", false
	}
	return hex.EncodeToString(bytes), true
	//
	//tp, err := ts.NewTxPayLoad()
	//if err != nil {
	//	return "", false
	//}
	//
	//msg, _ := hex.DecodeString(tp.ToBytesString())
	//
	//pubkey, _ := hex.DecodeString(ts.SenderPubkey)
	//
	//sig, err := hex.DecodeString(signature)
	//if err != nil || len(sig) != 64{
	//	return "", false
	//}
	//
	//if owcrypt.SUCCESS != owcrypt.Verify(pubkey, nil, msg, sig, owcrypt.ECC_CURVE_ED25519) {
	//	return "", false
	//}
	//
	//signned, err := ts.GetSignedTransaction(signature)
	//if err != nil {
	//	return "", false
	//}
	//
	//return signned, true
}

func GetBigIntAmountStr(amountStr string, amountDecimal int32) (string, error){
	d, err := decimal.NewFromString(amountStr)
	if err!= nil {
		return "", err
	}
	//atomDec := amountDec.Mul( decimal.)
	ten := math.BigPow(10, int64(amountDecimal) )
	w, err := decimal.NewFromString( ten.String() )
	if err!= nil {
		return "", err
	}

	d = d.Mul(w)
	return d.String(), nil
}