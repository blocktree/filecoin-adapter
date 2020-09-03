package filecoin_rpc

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/blocktree/filecoin-adapter/filecoinTransaction"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/specs-actors/actors/builtin"
	"github.com/filecoin-project/specs-actors/actors/crypto"
	"github.com/shopspring/decimal"
	"testing"
)

const (
	//baseURL = "http://127.0.0.1:1234/rpc/v0" //fil_test_local
	baseURL = "http://47.57.26.144:20031/rpc/v0" //fil_test_remote
	CidLength = 62
)

func TestGetCall(t *testing.T) {
	client := Client{BaseURL:baseURL}
	method := "Filecoin.ChainHead"
	//method := "Filecoin.ChainGetTipSetByHeight"
	//method := "Filecoin.ChainGetBlock"
	//method := "Filecoin.ChainGetBlockMessages"
	//method := "Filecoin.WalletBalance"
	//method := "Filecoin.ChainGetTipSet"
	//method := "Filecoin.StateGetReceipt"
	//method := "Filecoin.MpoolGetNonce"
	//method := "Filecoin.MpoolEstimateGasPrice"
	//method := "Filecoin.StateGetActor"

	//blockCids := make([]interface{}, 0)
	//tipSetKey := []interface{}{ blockCids }

	params := []interface{}{
		//"t1abuzgc6y4tirvo274gyayqvslfmgkua3ksuyu4y",
		//blockCids,
	}

	//for i := 0; i <= 10; i++ {
	result, err := client.Call(method, params)
	if err != nil {
		t.Logf("Get Call Result return: \n\t%+v\n", err)
	}

	//for _, block := range gjson.Get(result.Raw, "Blocks").Array() {
	//	height := uint64( gjson.Get(block.Raw, "Height").Uint() )
	//	fmt.Println(", height:", height )
	//}

	if result != nil {
		fmt.Println(method, ", result:", result.String() )
	}
	//}
}

func TestGetCallWithToken(t *testing.T) {
	client := Client{BaseURL:baseURL}
	//method := "Filecoin.ChainHead"
	//method := "Filecoin.MpoolPush"
	//method := "Filecoin.ChainGetTipSetByHeight"
	//method := "Filecoin.ChainGetBlock"
	//method := "Filecoin.ChainGetBlockMessages"
	//method := "Filecoin.WalletBalance"
	method := "Filecoin.MpoolPush"

	fromAddr, _ := address.NewFromString("t1xzefzapav6scdwhtt3dzbvihvqn5qx5tajgbzca" )
	toAddr, _ := address.NewFromString("t1wh2fhzvb5rcfoleedkupov442qp4hw34kzm52ki")

	//msg := filecoinTransaction.Message{
	//	To:       toAddr,
	//	From:     fromAddr,
	//	Value:    filecoinTransaction.NewInt(1),
	//	GasPrice: filecoinTransaction.NewInt(0),
	//	Nonce:   0,
	//	GasLimit: 1000000,
	//	Method:   builtin.MethodSend,
	//}

	valueStr, _ := GetBigIntAmountStr("1.35677", 18)
	value, _ := filecoinTransaction.BigFromString( valueStr )

	msg := filecoinTransaction.Message{
		To:       toAddr,
		From:     fromAddr,
		Value:    value,
		GasPrice: filecoinTransaction.NewInt(1),
		Nonce:    1,
		GasLimit: 1000000,
		Method:   builtin.MethodSend,
	}

	js, _ := json.Marshal(msg)
	fmt.Println("js : ", string(js))

	sigStr := "698c21f22a05a200a119521d240327b1574cb1c7edf5124f2e9c4ea407fada813cb3dd164713884dffea364d9818233549e22415835e772f94c180ccdc37d3a700"
	sigData, _ := hex.DecodeString(sigStr)
	sig := crypto.Signature{
		Type: 1,
		Data: sigData,
	}

	callMsg := map[string]interface{}{
		"message" :  msg,
		"signature" : sig,
	}

	params := []interface{}{ callMsg }

	//for i := 0; i <= 10; i++ {
		accessToken := "xxxxx"
		result, err := client.CallWithToken(accessToken, method, params)
		if err != nil {
			t.Logf("Get Call Result return: \n\t%+v\n", err)
		}

		//for _, block := range gjson.Get(result.Raw, "Blocks").Array() {
		//	height := uint64( gjson.Get(block.Raw, "Height").Uint() )
		//	fmt.Println(", height:", height )
		//}

		if result != nil {
			fmt.Println(method, ", result:", result.String() )
		}
	//}
}

// return : []{"/":"ba..."}...
func GetTipSetKey(hash string) []interface{}{
	blockCids := make([]interface{}, 0)
	for i:=0; i<(len(hash)/CidLength); i++ {
		blockCidHash := hash[i*CidLength:(i+1)*CidLength]

		blockCid := map[string]interface{}{
			"/" : blockCidHash,
		}

		blockCids = append(blockCids, blockCid)
	}
	return blockCids
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