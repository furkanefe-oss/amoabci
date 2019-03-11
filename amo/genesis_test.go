package amo

import (
	"encoding/hex"
	"os"
	"testing"

	"github.com/amolabs/tendermint-amo/crypto"
	"github.com/amolabs/tendermint-amo/crypto/p256"
	cmn "github.com/amolabs/tendermint-amo/libs/common"
	"github.com/amolabs/tendermint-amo/libs/db"
	"github.com/stretchr/testify/assert"

	"github.com/amolabs/amoabci/amo/store"
	"github.com/amolabs/amoabci/amo/types"
)

const testRoot = "genesis_test"

const (
	addr0Json = "7CECB223B976F27D77B0E03E95602DABCC28D876"
	t0json    = `{
	  "balances": [
		{
		  "owner": "7CECB223B976F27D77B0E03E95602DABCC28D876",
		  "amount": "100"
		}
	  ]
	}`
	t1json = `{
	  "balances": [
		{
		  "owner": "7CECB223B976F27D77B0E03E95602DABCC28D876",
		  "amount": "100"
		},
		{
		  "owner": "012F",
		  "amount": "10"
		}
	  ]
	}`
	t2json = `{
	  "balances": [
		{
		  "owner": "7CECB223B976F27D77B0E03E95602DABCC28D876",
		  "amount": "100"
		},
		{
		  "owner": "012F",
		  "amount": "10"
		}
	  ],
	  "parcels": []
	}`
)

func setupDB() {
	cmn.EnsureDir(testRoot, 0700)
}

func tearDownDB() {
	os.RemoveAll(testRoot)
}

func TestParseGenesisStateBytes(t *testing.T) {
	var bytes []byte

	stateBytes := []byte(t1json)
	genState, err := ParseGenesisStateBytes(stateBytes)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(genState.Balances))

	bytes, _ = hex.DecodeString(addr0Json)
	assert.Equal(t, crypto.Address(bytes), genState.Balances[0].Owner)
	assert.Equal(t, types.Currency(100), genState.Balances[0].Amount)

	// TODO: need to raise an error for this case
	bytes, _ = hex.DecodeString("012F")
	assert.Equal(t, crypto.Address(bytes), genState.Balances[1].Owner)
	assert.Equal(t, types.Currency(10), genState.Balances[1].Amount)

	// proper balances + garbage data
	stateBytes = []byte(t2json)
	genState, err = ParseGenesisStateBytes(stateBytes)
	// XXX: no member Parcels GenAmoAppState yet, but this should not raise an
	// error
	assert.NoError(t, err)

	bytes, _ = hex.DecodeString(addr0Json)
	assert.Equal(t, crypto.Address(bytes), genState.Balances[0].Owner)
	assert.Equal(t, types.Currency(100), genState.Balances[0].Amount)

}

func TestFillGenesisState(t *testing.T) {
	setupDB()

	s := store.NewStore(db.NewMemDB())

	// first fill the test store with some values
	addr1 := p256.GenPrivKey().PubKey().Address()
	addr2 := p256.GenPrivKey().PubKey().Address()
	s.SetBalance(addr1, types.Currency(10))
	s.SetBalance(addr2, types.Currency(20))

	assert.Equal(t, types.Currency(10), s.GetBalance(addr1))

	genState, err := ParseGenesisStateBytes([]byte(t0json))
	// this will purge previous data and fill with newly provided genesis state
	err = FillGenesisState(s, genState)
	assert.NoError(t, err)

	// check if the store has been purged prior to fill with genesis state
	assert.Equal(t, types.Currency(0), s.GetBalance(addr1))
	assert.Equal(t, types.Currency(0), s.GetBalance(addr2))

	// check if the genesis state is filled correctly
	addr0, _ := hex.DecodeString(addr0Json)
	assert.Equal(t, types.Currency(100), s.GetBalance(addr0))

	tearDownDB()
}