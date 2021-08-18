package tx

import (
	//"encoding/binary"
	//"encoding/json"
	"encoding/hex"
	"encoding/json"
	"strings"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/amolabs/amoabci/amo/code"
	"github.com/amolabs/amoabci/amo/store"
	"github.com/amolabs/amoabci/amo/types"
)

//// claim

type ClaimParamV6 struct {
	Target   string   `json:"target"`
	Document Document `json:"document"`
}

type Document struct {
	Context            string `json:"@context,omitempty"`
	Id                 string `json:"id"`
	Controller         string `json:"controller,omitempty"`
	VerificationMethod struct {
		Id           string `json:"id"`
		Type         string `json:"type"`
		Controller   string `json:"controller,omitempty"`
		PublicKeyJwk struct {
			Kty string `json:"kty"`
			Crv string `json:"crv"`
			X   string `json:"x"`
			Y   string `json:"y"`
		} `json:"publicKeyJwk"`
	} `json:"verificationMethod"`
	Authentication  string `json:"authentication"`
	AssertionMethod string `json:"assertionMethod,omitempty"`
}

func parseClaimParamV6(raw []byte) (ClaimParamV6, error) {
	var param ClaimParamV6
	err := json.Unmarshal(raw, &param)
	if err != nil {
		return param, err
	}
	return param, nil
}

type TxClaimV6 struct {
	TxBase
	Param ClaimParam `json:"-"`
}

var _ Tx = &TxClaimV6{}

func (t *TxClaimV6) Check() (uint32, string) {
	param, err := parseClaimParamV6(t.getPayload())
	if err != nil {
		return code.TxCodeBadParam, err.Error()
	}

	ss := strings.Split(param.Target, ":")
	if len(ss) != 3 || ss[0] != "did" || ss[1] != "amo" || len(ss[2]) != 40 {
		return code.TxCodeBadParam, "invalid target did"
	}
	_, err = hex.DecodeString(ss[2])
	if err != nil {
		return code.TxCodeBadParam, err.Error()
	}

	if param.Target != param.Document.Id {
		return code.TxCodeBadParam, "mismatching did"
	}

	return code.TxCodeOK, "ok"
}

func (t *TxClaimV6) Execute(store *store.Store) (uint32, string, []abci.Event) {
	txParam, err := parseClaimParamV6(t.getPayload())
	if err != nil {
		return code.TxCodeBadParam, err.Error(), nil
	}

	b, err := json.Marshal(txParam.Document)
	if err != nil {
		return code.TxCodeBadParam, err.Error(), nil
	}
	entry := &types.DIDEntry{
		Document: b,
	}
	store.SetDIDEntry(txParam.Target, entry)

	return code.TxCodeOK, "ok", []abci.Event{}
}

//// dismiss

type TxDismissV6 struct {
	TxBase
	Param DismissParam `json:"-"`
}

var _ Tx = &TxDismissV6{}

func (t *TxDismissV6) Check() (uint32, string) {
	_, err := parseDismissParam(t.getPayload())
	if err != nil {
		return code.TxCodeBadParam, err.Error()
	}

	return code.TxCodeOK, "ok"
}

func (t *TxDismissV6) Execute(store *store.Store) (uint32, string, []abci.Event) {
	txParam, err := parseClaimParam(t.getPayload())
	if err != nil {
		return code.TxCodeBadParam, err.Error(), nil
	}

	store.DeleteDIDEntry(txParam.Target)

	return code.TxCodeOK, "ok", []abci.Event{}
}
