package amo

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	tm "github.com/tendermint/tendermint/libs/common"

	"github.com/amolabs/amoabci/amo/code"
	"github.com/amolabs/amoabci/amo/store"
	"github.com/amolabs/amoabci/amo/types"
)

// MERKLE TREE SCOPE (IMPORTANT)
//   Query related funcs must get data from committed(saved) tree
//   NOT from working tree as the users or clients
//   SHOULD see the data which are already commited by validators
//   So, it is mandatory to use 'true' for 'committed' arg input
//   to query data from merkle tree

func queryAppConfig(config types.AMOAppConfig) (res abci.ResponseQuery) {
	jsonstr, _ := json.Marshal(config)
	res.Log = string(jsonstr)
	res.Key = []byte("config")
	res.Value = jsonstr
	res.Code = code.QueryCodeOK

	return
}

func queryBalance(s *store.Store, udc string, queryData []byte) (res abci.ResponseQuery) {
	if len(queryData) == 0 {
		res.Log = "error: no query_data"
		res.Code = code.QueryCodeNoKey
		return
	}

	var addr crypto.Address
	err := json.Unmarshal(queryData, &addr)
	if err != nil {
		res.Log = "error: unmarshal"
		res.Code = code.QueryCodeBadKey
		return
	}

	udcID := []byte(nil)
	if udc != "" {
		udcID, err = types.ConvIDFromStr(udc)
		if err != nil {
			res.Log = "error: cannot convert udc id"
			res.Code = code.QueryCodeBadKey
			return
		}
	}

	bal := s.GetUDCBalance(udcID, addr, true)

	jsonstr, _ := json.Marshal(bal)
	res.Log = string(jsonstr)
	// XXX: tendermint will convert this using base64 encoding
	res.Value = jsonstr
	res.Code = code.QueryCodeOK
	res.Key = queryData

	return
}

func queryStake(s *store.Store, queryData []byte) (res abci.ResponseQuery) {
	if len(queryData) == 0 {
		res.Log = "error: no query_data"
		res.Code = code.QueryCodeNoKey
		return
	}

	var addr crypto.Address
	err := json.Unmarshal(queryData, &addr)
	if err != nil {
		res.Log = "error: unmarshal"
		res.Code = code.QueryCodeBadKey
		return
	}

	stake := s.GetStake(addr, true)
	if stake == nil {
		res.Log = "error: no stake"
		res.Code = code.QueryCodeNoMatch
		return
	}

	stakeEx := types.StakeEx{stake, s.GetDelegatesByDelegatee(addr, true)}
	jsonstr, _ := json.Marshal(stakeEx)
	res.Log = string(jsonstr)
	res.Value = jsonstr
	res.Code = code.QueryCodeOK
	res.Key = queryData

	return
}

func queryDelegate(s *store.Store, queryData []byte) (res abci.ResponseQuery) {
	if len(queryData) == 0 {
		res.Log = "error: no query_data"
		res.Code = code.QueryCodeNoKey
		return
	}

	var addr crypto.Address
	err := json.Unmarshal(queryData, &addr)
	if err != nil {
		res.Log = "error: unmarshal"
		res.Code = code.QueryCodeBadKey
		return
	}

	delegate := s.GetDelegate(addr, true)
	if delegate == nil {
		res.Log = "error: no delegate"
		res.Code = code.QueryCodeNoMatch
		return
	}

	jsonstr, _ := json.Marshal(delegate)
	res.Log = string(jsonstr)
	res.Value = jsonstr
	res.Code = code.QueryCodeOK
	res.Key = queryData

	return
}

func queryValidator(s *store.Store, queryData []byte) (res abci.ResponseQuery) {
	if len(queryData) == 0 {
		res.Log = "error: no query_data"
		res.Code = code.QueryCodeNoKey
		return
	}

	var addr crypto.Address
	err := json.Unmarshal(queryData, &addr)
	if err != nil {
		res.Log = "error: unmarshal"
		res.Code = code.QueryCodeBadKey
		return
	}

	holder := s.GetHolderByValidator(addr, true)
	jsonstr, _ := json.Marshal(crypto.Address(holder))
	res.Log = string(jsonstr)
	res.Value = jsonstr
	res.Code = code.QueryCodeOK
	res.Key = queryData

	return
}

func queryStorage(s *store.Store, queryData []byte) (res abci.ResponseQuery) {
	if len(queryData) == 0 {
		res.Log = "error: no query_data"
		res.Code = code.QueryCodeNoKey
		return
	}

	var rawID uint32
	err := json.Unmarshal(queryData, &rawID)
	if err != nil {
		res.Log = "error: unmarshal"
		res.Code = code.QueryCodeBadKey
		return
	}

	storageID := types.ConvIDFromUint(rawID)
	storage := s.GetStorage(storageID, true)
	if storage == nil {
		res.Log = "error: no such storage"
		res.Code = code.QueryCodeNoMatch
		return
	}

	jsonstr, _ := json.Marshal(storage)
	res.Log = string(jsonstr)
	res.Value = jsonstr
	res.Code = code.QueryCodeOK
	res.Key = queryData

	return
}

func queryDraft(s *store.Store, queryData []byte) (res abci.ResponseQuery) {
	if len(queryData) == 0 {
		res.Log = "error: no query_data"
		res.Code = code.QueryCodeNoKey
		return
	}

	var rawID uint32
	err := json.Unmarshal(queryData, &rawID)
	if err != nil {
		res.Log = "error: unmarshal"
		res.Code = code.QueryCodeBadKey
		return
	}

	draftID := types.ConvIDFromUint(rawID)
	draft := s.GetDraft(draftID, true)
	if draft == nil {
		res.Log = "error: no draft"
		res.Code = code.QueryCodeNoMatch
		return
	}

	draftEx := types.DraftEx{
		Draft: draft,
		Votes: s.GetVotes(draftID, true),
	}

	jsonstr, err := json.Marshal(draftEx)
	if err != nil {
		res.Log = "error: marshal"
		res.Code = code.QueryCodeBadKey
		return
	}

	res.Log = string(jsonstr)
	res.Value = jsonstr
	res.Code = code.QueryCodeOK
	res.Key = queryData

	return
}

func queryVote(s *store.Store, queryData []byte) (res abci.ResponseQuery) {
	if len(queryData) == 0 {
		res.Log = "error: no query_data"
		res.Code = code.QueryCodeNoKey
		return
	}

	var param struct {
		DraftID uint32         `json:"draft_id"`
		Voter   crypto.Address `json:"voter"`
	}

	err := json.Unmarshal(queryData, &param)
	if err != nil {
		res.Log = "error: unmarshal"
		res.Code = code.QueryCodeBadKey
		return
	}

	draftID := types.ConvIDFromUint(param.DraftID)
	if len(param.Voter) != crypto.AddressSize {
		res.Log = "error: not avaiable address"
		res.Code = code.QueryCodeBadKey
		return
	}

	vote := s.GetVote(draftID, param.Voter, true)
	if vote == nil {
		res.Log = "error: no vote"
		res.Code = code.QueryCodeNoMatch
		return
	}

	voteInfo := types.VoteInfo{
		Voter: param.Voter,
		Vote:  vote,
	}

	jsonstr, _ := json.Marshal(voteInfo)
	res.Log = string(jsonstr)
	res.Value = jsonstr
	res.Code = code.QueryCodeOK
	res.Key = queryData

	return
}

func queryParcel(s *store.Store, queryData []byte) (res abci.ResponseQuery) {
	if len(queryData) == 0 {
		res.Log = "error: no query_data"
		res.Code = code.QueryCodeNoKey
		return
	}

	var id tm.HexBytes
	err := json.Unmarshal(queryData, &id)
	if err != nil {
		res.Log = "error: unmarshal"
		res.Code = code.QueryCodeBadKey
		return
	}

	parcel := s.GetParcel(id, true)
	if parcel == nil {
		res.Log = "error: no such parcel"
		res.Code = code.QueryCodeNoMatch
		return
	}

	parcelEx := types.ParcelEx{
		Parcel:   parcel,
		Requests: s.GetRequests(id, true),
		Usages:   s.GetUsages(id, true),
	}

	jsonstr, _ := json.Marshal(parcelEx)
	res.Log = string(jsonstr)
	res.Value = jsonstr
	res.Code = code.QueryCodeOK
	res.Key = queryData

	return
}

func queryRequest(s *store.Store, queryData []byte) (res abci.ResponseQuery) {
	if len(queryData) == 0 {
		res.Log = "error: no query_data"
		res.Code = code.QueryCodeNoKey
		return
	}

	keyMap := make(map[string]tm.HexBytes)
	err := json.Unmarshal(queryData, &keyMap)
	if err != nil {
		res.Log = "error: unmarshal"
		res.Code = code.QueryCodeBadKey
		return
	}
	if _, ok := keyMap["buyer"]; !ok {
		res.Log = "error: buyer is missing"
		res.Code = code.QueryCodeBadKey
		return
	}
	if _, ok := keyMap["target"]; !ok {
		res.Log = "error: target is missing"
		res.Code = code.QueryCodeBadKey
		return
	}
	addr := crypto.Address(keyMap["buyer"])
	if len(addr) != crypto.AddressSize {
		res.Log = "error: not avaiable address"
		res.Code = code.QueryCodeBadKey
		return
	}

	// TODO: parse parcel id
	parcelID := keyMap["target"]

	request := s.GetRequest(addr, parcelID, true)
	if request == nil {
		res.Log = "error: no request"
		res.Code = code.QueryCodeNoMatch
		return
	}

	jsonstr, _ := json.Marshal(request)
	res.Log = string(jsonstr)
	res.Value = jsonstr
	res.Code = code.QueryCodeOK
	res.Key = queryData

	return
}

func queryUsage(s *store.Store, queryData []byte) (res abci.ResponseQuery) {
	if len(queryData) == 0 {
		res.Log = "error: no query_data"
		res.Code = code.QueryCodeNoKey
		return
	}

	keyMap := make(map[string]tm.HexBytes)
	err := json.Unmarshal(queryData, &keyMap)
	if err != nil {
		res.Log = "error: unmarshal"
		res.Code = code.QueryCodeBadKey
		return
	}
	if _, ok := keyMap["buyer"]; !ok {
		res.Log = "error: buyer is missing"
		res.Code = code.QueryCodeBadKey
		return
	}
	if _, ok := keyMap["target"]; !ok {
		res.Log = "error: target is missing"
		res.Code = code.QueryCodeBadKey
		return
	}
	addr := crypto.Address(keyMap["buyer"])
	if len(addr) != crypto.AddressSize {
		res.Log = "error: not avaiable address"
		res.Code = code.QueryCodeBadKey
		return
	}

	// TODO: parse parcel id
	parcelID := keyMap["target"]

	usage := s.GetUsage(addr, parcelID, true)
	if usage == nil {
		res.Log = "error: no usage"
		res.Code = code.QueryCodeNoMatch
		return
	}

	jsonstr, _ := json.Marshal(usage)
	res.Log = string(jsonstr)
	res.Value = jsonstr
	res.Code = code.QueryCodeOK
	res.Key = queryData

	return
}

func queryBlockIncentives(s *store.Store, queryData []byte) (res abci.ResponseQuery) {
	var (
		incentives []store.IncentiveInfo
		tmp        string
		height     int64
	)

	if len(queryData) == 0 {
		res.Log = "error: no query_data"
		res.Code = code.QueryCodeNoKey
		return
	}

	err := json.Unmarshal(queryData, &tmp)
	if err != nil {
		res.Log = "error: unmarshal"
		res.Code = code.QueryCodeNoKey
		return
	}

	height, err = strconv.ParseInt(tmp, 10, 64)
	if err != nil {
		res.Log = "error: cannot convert string to int64"
		res.Code = code.QueryCodeNoKey
		return
	}

	incentives = s.GetBlockIncentiveRecords(height)
	if len(incentives) == 0 {
		res.Log = fmt.Sprintf("no match: %d", height)
		res.Code = code.QueryCodeNoMatch
		return
	}

	jsonstr, _ := json.Marshal(incentives)
	res.Log = string(jsonstr)
	res.Value = jsonstr
	res.Code = code.QueryCodeOK
	res.Key = queryData

	return
}

func queryAddressIncentives(s *store.Store, queryData []byte) (res abci.ResponseQuery) {
	var (
		incentives []store.IncentiveInfo
		address    crypto.Address
	)

	if len(queryData) == 0 {
		res.Log = "error: no query_data"
		res.Code = code.QueryCodeNoKey
		return
	}

	err := json.Unmarshal(queryData, &address)
	if err != nil {
		res.Log = "error: unmarshal"
		res.Code = code.QueryCodeNoKey
		return
	}

	incentives = s.GetAddressIncentiveRecords(address)
	if len(incentives) == 0 {
		res.Log = "error: no incentives"
		res.Code = code.QueryCodeNoMatch
		return
	}

	jsonstr, _ := json.Marshal(incentives)
	res.Log = string(jsonstr)
	res.Value = jsonstr
	res.Code = code.QueryCodeOK
	res.Key = queryData

	return
}

func queryIncentive(s *store.Store, queryData []byte) (res abci.ResponseQuery) {
	var (
		incentives []store.IncentiveInfo
		height     int64
		address    crypto.Address
	)

	if len(queryData) == 0 {
		res.Log = "error: no query_data"
		res.Code = code.QueryCodeNoKey
		return
	}

	var keys struct {
		Height  string         `json:"height"`
		Address crypto.Address `json:"address"`
	}

	err := json.Unmarshal(queryData, &keys)
	if err != nil {
		res.Log = "error: unmarshal"
		res.Code = code.QueryCodeBadKey
		return
	}

	height, err = strconv.ParseInt(keys.Height, 10, 64)
	if err != nil {
		res.Log = fmt.Sprintf("error: cannot convert %s", keys.Height)
		res.Code = code.QueryCodeBadKey
		return
	}

	address = keys.Address

	incentive := s.GetIncentiveRecord(height, address)
	if reflect.DeepEqual(incentive, store.IncentiveInfo{}) {
		res.Log = "error: no incentives"
		res.Code = code.QueryCodeNoMatch
		return
	}

	incentives = append(incentives, incentive)

	jsonstr, _ := json.Marshal(incentives)
	res.Log = string(jsonstr)
	res.Value = jsonstr
	res.Code = code.QueryCodeOK
	res.Key = queryData

	return
}
