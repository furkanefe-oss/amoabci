package amo

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	tm "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/libs/log"
	tmdb "github.com/tendermint/tm-db"

	"github.com/amolabs/amoabci/amo/code"
	astore "github.com/amolabs/amoabci/amo/store"
	"github.com/amolabs/amoabci/amo/tx"
	"github.com/amolabs/amoabci/amo/types"
)

var (
	stateKey = []byte("stateKey") // TODO: remove this when applying merkle tree
)

const (
	// versions
	AMOAppVersion      = "v1.1.0-dev"
	AMOProtocolVersion = 0x2
	// hard-coded configs
	defaultMaxValidators   = 100
	defaultWeightValidator = int64(2)
	defaultWeightDelegator = int64(1)
	defaultBlkReward       = uint64(0)
	defaultTxReward        = uint64(types.OneAMOUint64 / 10)
	defaultLockupPeriod    = uint64(1000000)
)

// Output are sorted by voting power.
func findValUpdates(oldVals, newVals abci.ValidatorUpdates) abci.ValidatorUpdates {
	sort.Slice(oldVals, func(i, j int) bool {
		return bytes.Compare(oldVals[i].PubKey.Data, oldVals[j].PubKey.Data) < 0
	})
	sort.Slice(newVals, func(i, j int) bool {
		return bytes.Compare(newVals[i].PubKey.Data, newVals[j].PubKey.Data) < 0
	})

	// extract updates
	i := 0
	j := 0
	updates := abci.ValidatorUpdates{}
	for i < len(oldVals) && j < len(newVals) {
		comp := bytes.Compare(oldVals[i].PubKey.Data, newVals[j].PubKey.Data)
		if comp < 0 {
			updates = append(updates, abci.ValidatorUpdate{
				PubKey: oldVals[i].PubKey, Power: 0})
			i++
		} else if comp == 0 {
			updates = append(updates, newVals[j])
			i++
			j++
		} else {
			updates = append(updates, newVals[j])
			j++
		}
	}

	for ; i < len(oldVals); i++ {
		updates = append(updates, abci.ValidatorUpdate{
			PubKey: oldVals[i].PubKey, Power: 0})
	}

	for ; j < len(newVals); j++ {
		updates = append(updates, newVals[j])
	}

	sort.Slice(updates, func(i, j int) bool {
		// reverse order
		return updates[i].Power > updates[j].Power
	})
	return updates
}

// TODO: use 2-stage state
type State struct {
	//Walk          int64 `json:"-"` // TODO: remove this
	merkleVersion int64 `json:"merkle_version"`
	height        int64 `json:"-"` // current block height
	appHash       []byte
	lastHeight    int64  `json:"last_height"`   // last completed block height
	lastAppHash   []byte `json:"last_app_hash"` // TODO: use merkle tree
}

type AMOAppConfig struct {
	MaxValidators   uint64
	WeightValidator int64
	WeightDelegator int64
	BlkReward       uint64
	TxReward        uint64
	LockupPeriod    uint64
}

type AMOApp struct {
	// app scaffold
	abci.BaseApplication
	logger log.Logger

	// app config
	config AMOAppConfig

	// internal mekrle tree
	merkleDB tmdb.DB

	// internal state
	stateDB tmdb.DB
	indexDB tmdb.DB
	state   State
	store   *astore.Store

	// runtime temporary variables
	doValUpdate bool
	oldVals     abci.ValidatorUpdates
}

var _ abci.Application = (*AMOApp)(nil)

func NewAMOApp(mdb tmdb.DB, sdb tmdb.DB, idb tmdb.DB, l log.Logger) *AMOApp {
	if l == nil {
		l = log.NewNopLogger()
	}
	if mdb == nil {
		mdb = tmdb.NewMemDB()
	}
	if sdb == nil {
		sdb = tmdb.NewMemDB()
	}
	if idb == nil {
		idb = tmdb.NewMemDB()
	}

	app := &AMOApp{
		logger: l,
		config: AMOAppConfig{ // TODO: read from config file
			defaultMaxValidators,
			defaultWeightValidator,
			defaultWeightDelegator,
			defaultBlkReward,
			defaultTxReward,
			defaultLockupPeriod,
		},
		merkleDB: mdb,
		stateDB:  sdb,
		indexDB:  idb,
		store:    astore.NewStore(mdb, sdb, idb),
	}
	// TODO: use something more elegant
	tx.ConfigLockupPeriod = app.config.LockupPeriod
	app.load()
	return app
}

func (app *AMOApp) load() {
	stateBytes := app.stateDB.Get(stateKey)
	if len(stateBytes) != 0 {
		err := json.Unmarshal(stateBytes, &app.state)
		if err != nil {
			panic(err)
		}
	}
}

func (app *AMOApp) save() {
	stateBytes, err := json.Marshal(app.state)
	if err != nil {
		panic(err)
	}
	app.stateDB.Set(stateKey, stateBytes)
}

func (app *AMOApp) Info(req abci.RequestInfo) (resInfo abci.ResponseInfo) {
	return abci.ResponseInfo{
		Data:             fmt.Sprintf("{\"merkle_version\":%v}", app.state.merkleVersion),
		Version:          AMOAppVersion,
		AppVersion:       AMOProtocolVersion,
		LastBlockHeight:  app.state.lastHeight,
		LastBlockAppHash: app.state.lastAppHash,
	}
}

func (app *AMOApp) InitChain(req abci.RequestInitChain) abci.ResponseInitChain {
	genAppState, err := ParseGenesisStateBytes(req.AppStateBytes)
	// TODO: use proper methods to inform error
	if err != nil {
		return abci.ResponseInitChain{}
	}
	if FillGenesisState(app.store, genAppState) != nil {
		return abci.ResponseInitChain{}
	}

	/*
		app.state.Walk = 0 // TODO: Replace this with merkle tree
		b := make([]byte, 8)
		binary.PutVarint(b, app.state.Walk)
		app.state.lastHeight = 0
		app.state.lastAppHash = b
	*/

	hash, version, err := app.store.Save()
	if err != nil {
		return abci.ResponseInitChain{}
	}

	app.state.merkleVersion = version
	app.state.lastHeight = 0
	app.state.lastAppHash = hash

	app.save()
	app.logger.Info("InitChain: new genesis app state applied.")

	return abci.ResponseInitChain{
		Validators: app.store.GetValidators(app.config.MaxValidators),
	}
}

// TODO: return proof also
func (app *AMOApp) Query(reqQuery abci.RequestQuery) (resQuery abci.ResponseQuery) {
	switch reqQuery.Path {
	case "/balance":
		resQuery = queryBalance(app.store, reqQuery.Data)
	case "/stake":
		resQuery = queryStake(app.store, reqQuery.Data)
	case "/delegate":
		resQuery = queryDelegate(app.store, reqQuery.Data)
	case "/validator":
		resQuery = queryValidator(app.store, reqQuery.Data)
	case "/parcel":
		resQuery = queryParcel(app.store, reqQuery.Data)
	case "/request":
		resQuery = queryRequest(app.store, reqQuery.Data)
	case "/usage":
		resQuery = queryUsage(app.store, reqQuery.Data)
	default:
		resQuery.Code = code.QueryCodeBadPath
	}

	app.logger.Debug("Query: "+reqQuery.Path, "query_data", reqQuery.Data)

	return resQuery
}

func (app *AMOApp) BeginBlock(req abci.RequestBeginBlock) (res abci.ResponseBeginBlock) {
	app.state.height = req.Header.Height
	app.doValUpdate = false
	app.oldVals = app.store.GetValidators(app.config.MaxValidators)

	proposer := req.Header.GetProposerAddress()
	staker := app.store.GetHolderByValidator(proposer)
	numTxs := req.Header.GetNumTxs()

	// XXX no means to convey error to res
	app.DistributeReward(staker, numTxs)

	return res
}

// Invariant checks. Do not consider app's store.
// - check signature
// - check parameter format
func (app *AMOApp) CheckTx(req abci.RequestCheckTx) abci.ResponseCheckTx {
	t, err := tx.ParseTx(req.Tx)
	if err != nil {
		return abci.ResponseCheckTx{
			Code:      code.TxCodeBadParam,
			Info:      err.Error(),
			Codespace: "amo",
		}
	}
	if !t.Verify() {
		return abci.ResponseCheckTx{
			Code:      code.TxCodeBadSignature,
			Info:      "Signature verification failed",
			Codespace: "amo",
		}
	}

	rc, info := t.Check()

	return abci.ResponseCheckTx{
		Code:      rc,
		Info:      info,
		Codespace: "amo",
	}
}

func (app *AMOApp) DeliverTx(req abci.RequestDeliverTx) abci.ResponseDeliverTx {
	t, err := tx.ParseTx(req.Tx)
	if err != nil {
		return abci.ResponseDeliverTx{
			Code:      code.TxCodeBadParam,
			Info:      err.Error(),
			Codespace: "amo",
		}
	}

	tags := []tm.KVPair{
		{Key: []byte("tx.type"), Value: []byte(t.GetType())},
		{Key: []byte("tx.sender"), Value: []byte(t.GetSender().String())},
	}

	rc, info, opTags := t.Execute(app.store)

	// if the operation was not successful, change nothing
	if rc == code.TxCodeOK {
		if t.GetType() == "stake" || t.GetType() == "withdraw" ||
			t.GetType() == "delegate" || t.GetType() == "retract" {
			app.doValUpdate = true
		}
		tags = append(tags, opTags...)
	}

	return abci.ResponseDeliverTx{
		Code: rc,
		Info: info,
		Events: []abci.Event{abci.Event{
			Type:       "default",
			Attributes: tags,
		}},
		Codespace: "amo",
	}
}

// TODO: use req.Height
func (app *AMOApp) EndBlock(req abci.RequestEndBlock) (res abci.ResponseEndBlock) {
	if app.doValUpdate {
		app.doValUpdate = false
		newVals := app.store.GetValidators(app.config.MaxValidators)
		res.ValidatorUpdates = findValUpdates(app.oldVals, newVals)
	}
	app.store.LoosenLockedStakes()

	// update appHash
	hash := app.store.Root()
	if hash == nil {
		return abci.ResponseEndBlock{}
	}

	app.state.appHash = hash

	return res
}

func (app *AMOApp) Commit() abci.ResponseCommit {
	hash, version, err := app.store.Save()
	if err != nil {
		return abci.ResponseCommit{}
	}

	ok := bytes.Equal(hash, app.state.appHash)
	if !ok {
		return abci.ResponseCommit{}
	}

	app.state.merkleVersion = version
	app.state.lastAppHash = app.state.appHash
	app.state.lastHeight = app.state.height

	app.save()

	return abci.ResponseCommit{Data: app.state.lastAppHash}
}

/////////////////////////////////////

func (app *AMOApp) DistributeReward(staker crypto.Address, numTxs int64) error {
	stake := app.store.GetStake(staker)
	if stake == nil {
		return errors.New("No stake, no reward.")
	}
	ds := app.store.GetDelegatesByDelegatee(staker)

	var tmp, tmp2 types.Currency

	// total reward
	var rTotal, rTx types.Currency
	rTotal.Set(app.config.BlkReward)
	rTx.Set(app.config.TxReward)
	tmp.SetInt64(numTxs)
	tmp.Mul(&tmp.Int, &rTx.Int)
	rTotal.Add(&tmp)

	// weighted sum
	var wsum, w big.Int
	w.SetInt64(app.config.WeightValidator)
	wsum.Mul(&w, &stake.Amount.Int)
	w.SetInt64(app.config.WeightDelegator)
	for _, d := range ds {
		tmp.Mul(&w, &d.Amount.Int)
		wsum.Add(&wsum, &tmp.Int)
	}
	// individual rewards
	tmp.Set(0) // subtotal for delegate holders
	for _, d := range ds {
		tmp2 = *partialReward(app.config.WeightDelegator, &d.Amount.Int, &wsum, &rTotal)
		tmp.Add(&tmp2) // update subtotal
		b := app.store.GetBalance(d.Delegator).Add(&tmp2)
		app.store.SetBalance(d.Delegator, b) // update balance
		app.logger.Debug("Block reward",
			"delegate", hex.EncodeToString(d.Delegator), "reward", tmp2.Int64())
	}
	tmp2.Int.Sub(&rTotal.Int, &tmp.Int) // calc validator reward
	b := app.store.GetBalance(staker).Add(&tmp2)
	app.store.SetBalance(staker, b)
	app.logger.Debug("Block reward",
		"proposer", hex.EncodeToString(staker)[:20], "reward", tmp2.Int64())

	return nil
}

/////////////////////////////////////

// r = (weight * stake / total) * base
// TODO: eliminate ambiguity in float computation
func partialReward(weight int64, stake, total *big.Int, base *types.Currency) *types.Currency {
	var wf, t1f, t2f big.Float
	wf.SetInt64(weight)
	t1f.SetInt(stake)
	t1f.Mul(&wf, &t1f)
	t2f.SetInt(total)
	t1f.Quo(&t1f, &t2f)
	t2f.SetInt(&base.Int)
	t1f.Mul(&t1f, &t2f)
	r := types.Currency{}
	t1f.Int(&r.Int)
	return &r
}
