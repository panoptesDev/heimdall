package topup_test

import (
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/client/context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ethTypes "github.com/maticnetwork/bor/core/types"
	"github.com/maticnetwork/heimdall/contracts/stakinginfo"
	"github.com/maticnetwork/heimdall/topup/types"
	hmTypes "github.com/maticnetwork/heimdall/types"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	"github.com/maticnetwork/heimdall/app"
	"github.com/maticnetwork/heimdall/helper/mocks"
	"github.com/maticnetwork/heimdall/topup"
	"github.com/maticnetwork/heimdall/types/simulation"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// HandlerTestSuite integrate test suite context object
type HandlerTestSuite struct {
	suite.Suite

	app            *app.HeimdallApp
	ctx            sdk.Context
	cliCtx         context.CLIContext
	querier        sdk.Querier
	handler        sdk.Handler
	contractCaller mocks.IContractCaller
}

// SetupTest setup all necessary things for querier tesing
func (suite *HandlerTestSuite) SetupTest() {
	suite.app, suite.ctx, suite.cliCtx = createTestApp(false)
	suite.contractCaller = mocks.IContractCaller{}
	suite.querier = topup.NewQuerier(suite.app.TopupKeeper, &suite.contractCaller)
	suite.handler = topup.NewHandler(suite.app.TopupKeeper, &suite.contractCaller)
}

// TestHandlerTestSuite
func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}

func (suite *HandlerTestSuite) TestHandleMsgTopup() {
	t, app, ctx := suite.T(), suite.app, suite.ctx

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	txHash := hmTypes.HexToHeimdallHash("123")
	index := simulation.RandIntBetween(r1, 0, 100)
	logIndex := uint64(index)

	pAddress := hmTypes.HexToHeimdallAddress("123")
	validatorId := uint64(simulation.RandIntBetween(r1, 0, 100))

	chainParams := app.ChainKeeper.GetParams(ctx)
	txreceipt := &ethTypes.Receipt{
		BlockNumber: big.NewInt(10),
		// Index:       uint(index),
		// Address:     chainParams.ChainParams.StakingInfoAddress.EthAddress(),
	}

	msgTopup := types.NewMsgTopup(pAddress, validatorId, txHash, logIndex)

	stakinginfoTopUpFee := &stakinginfo.StakinginfoTopUpFee{
		ValidatorId: new(big.Int).SetUint64(validatorId),
		Signer:      pAddress.EthAddress(),
		Fee:         big.NewInt(0),
	}

	suite.contractCaller.On("GetConfirmedTxReceipt", mock.Anything, txHash.EthHash(), chainParams.TxConfirmationTime).Return(txreceipt, nil)

	suite.contractCaller.On("DecodeValidatorTopupFeesEvent", chainParams.ChainParams.StakingInfoAddress.EthAddress(), mock.Anything, msgTopup.LogIndex).Return(stakinginfoTopUpFee, nil)
	result := topup.HandleMsgTopup(ctx, app.TopupKeeper, msgTopup, &suite.contractCaller)
	// TODO: send coin error {10 sdk [] {"codespace":"sdk","code":10,"message":"insufficient account funds; 100000000matic < 1000000000000000matic"} 0 0 []}
	require.True(t, result.IsOK(), "expected topup to be done, got %v", result)
}

func (suite *HandlerTestSuite) TestHandleMsgWithdrawFee() {
	t, app, ctx := suite.T(), suite.app, suite.ctx
	amount, _ := big.NewInt(0).SetString("0", 10)
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	validatorId := uint64(simulation.RandIntBetween(r1, 0, 100))

	privKey1 := secp256k1.GenPrivKey()
	pubkey := hmTypes.NewPubKey(privKey1.PubKey().Bytes())
	validatorAddress := pubkey.Address()

	startBlock := uint64(simulation.RandIntBetween(r1, 1, 100))

	power := simulation.RandIntBetween(r1, 1, 100)

	timeAlive := uint64(10)

	newVal := hmTypes.Validator{
		ID:               hmTypes.NewValidatorID(validatorId),
		StartEpoch:       startBlock,
		EndEpoch:         startBlock + timeAlive,
		VotingPower:      int64(power),
		Signer:           hmTypes.HexToHeimdallAddress(pubkey.Address().String()),
		PubKey:           pubkey,
		ProposerPriority: 0,
	}
	app.StakingKeeper.AddValidator(ctx, newVal)

	msgWithdrawFee := types.NewMsgWithdrawFee(
		hmTypes.BytesToHeimdallAddress(validatorAddress.Bytes()),
		sdk.NewIntFromBigInt(amount),
	)
	result := topup.HandleMsgWithdrawFee(ctx, app.TopupKeeper, msgWithdrawFee)

	require.True(t, result.IsOK(), "expected withdraw tobe done, git %v", result)
}
