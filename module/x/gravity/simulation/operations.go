package simulation

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/peggyjv/gravity-bridge/module/v2/x/gravity/keeper"
	"github.com/peggyjv/gravity-bridge/module/v2/x/gravity/types"
)

// Simulation operation weights constants
//
//nolint:gosec // these are not hardcoded credentials.
const (
	OpWeightMsgDeleteKeys                   = "op_weight_delegate_keys"
	OpWeightMsgSubmitEthereumTxConfirmation = "op_weight_msg_submit_ethereum_tx_confirmation"
	OpWeightMsgSubmitEthereumEvent          = "op_weight_msg_submit_ethereum_event"
	OpWeightMsgSendToEthereum               = "op_weight_msg_send_to_ethereum"
	OpWeightMsgRequestBatchTx               = "op_weight_msg_request_batch_tx"
	OpWeightMsgCancelSendToEthereum         = "op_weight_msg_cancel_send_to_ethereum"
	OpWeightMsgEthereumHeightVote           = "op_weight_msg_ethereum_height_vote"

	DefaultWeight  = 50
	maxWaitSeconds = 10
)

// Gravity message types and routes
var (
	TypeMsgDeleteKeys                   = sdk.MsgTypeURL(&types.MsgDelegateKeys{})
	TypeMsgSubmitEthereumTxConfirmation = sdk.MsgTypeURL(&types.MsgSubmitEthereumTxConfirmation{})
	TypeMsgSubmitEthereumEvent          = sdk.MsgTypeURL(&types.MsgSubmitEthereumEvent{})
	TypeMsgMsgSendToEthereum            = sdk.MsgTypeURL(&types.MsgSendToEthereum{})
	TypeMsgRequestBatchTx               = sdk.MsgTypeURL(&types.MsgRequestBatchTx{})
	TypeMsgCancelSendToEthereum         = sdk.MsgTypeURL(&types.MsgCancelSendToEthereum{})
	TypeMsgEthereumHeightVote           = sdk.MsgTypeURL(&types.MsgEthereumHeightVote{})
)

type simulateContext struct {
	gk  keeper.Keeper
	ak  types.AccountKeeper
	bk  types.BankKeeper
	cdc codec.JSONCodec
}

type opGenerator func(ctx *simulateContext) simtypes.Operation

// WeightedOperations returns all the operations from the module with their respective weights
func WeightedOperations(
	appParams simtypes.AppParams, cdc codec.JSONCodec, gk keeper.Keeper, ak types.AccountKeeper, bk types.BankKeeper,
) simulation.WeightedOperations {
	var ops simulation.WeightedOperations
	opFuncs := map[string]opGenerator{
		OpWeightMsgDeleteKeys:                   SimulateMsgDeleteKeys,
		OpWeightMsgSubmitEthereumTxConfirmation: SimulateMsgSubmitEthereumTxConfirmation,
		OpWeightMsgSubmitEthereumEvent:          SimulateMsgSubmitEthereumEvent,
		OpWeightMsgSendToEthereum:               SimulateMsgSendToEthereum,
		OpWeightMsgRequestBatchTx:               SimulateMsgRequestBatchTx,
		OpWeightMsgCancelSendToEthereum:         SimulateMsgCancelSendToEthereum,
		OpWeightMsgEthereumHeightVote:           SimulateMsgEthereumHeightVote,
	}

	ctx := &simulateContext{
		gk:  gk,
		ak:  ak,
		bk:  bk,
		cdc: cdc,
	}

	for k, f := range opFuncs {
		var v int
		appParams.GetOrGenerate(cdc, k, &v, nil,
			func(_ *rand.Rand) {
				v = DefaultWeight
			},
		)
		ops = append(
			ops,
			simulation.NewWeightedOperation(
				v,
				f(ctx),
			),
		)
	}
	return ops
}

func SimulateMsgDeleteKeys(simCtx *simulateContext) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		ak := simCtx.ak
		bk := simCtx.bk
		sk := simCtx.gk.StakingKeeper
		cdc := simCtx.cdc

		val, err := randomValidator(ctx, r, sk, accs)
		if err != nil {
			return simtypes.OperationMsg{}, nil, fmt.Errorf("none of the sim account is a validator")
		}
		valAddr := sdk.ValAddress(val.Address).String()
		orchAddr := val.Address.String()
		ethAddr := common.BytesToAddress(val.Address.Bytes()).String()
		nonce, err := ak.GetSequence(ctx, val.Address)
		if err != nil {
			return simtypes.OperationMsg{}, nil, err
		}

		bz := cdc.MustMarshalJSON(&types.DelegateKeysSignMsg{
			ValidatorAddress: valAddr,
			Nonce:            nonce,
		})

		hash := crypto.Keccak256Hash(bz).Bytes()
		ethPrivKey, err := crypto.ToECDSA(val.PrivKey.Bytes())
		if err != nil {
			return simtypes.OperationMsg{}, nil, err
		}
		sig, err := types.NewEthereumSignature(hash, ethPrivKey)
		if err != nil {
			return simtypes.OperationMsg{}, nil, err
		}

		msg := &types.MsgDelegateKeys{
			ValidatorAddress:    valAddr,
			OrchestratorAddress: orchAddr,
			EthereumAddress:     ethAddr,
			EthSignature:        sig,
		}

		txCtx := simulation.OperationInput{
			R:               r,
			App:             app,
			TxGen:           simappparams.MakeTestEncodingConfig().TxConfig,
			Cdc:             nil,
			Msg:             msg,
			MsgType:         msg.Type(),
			Context:         ctx,
			SimAccount:      val,
			AccountKeeper:   ak,
			Bankkeeper:      bk,
			ModuleName:      types.ModuleName,
			CoinsSpentInMsg: sdk.NewCoins(),
		}

		oper, ops, err := simulation.GenAndDeliverTxWithRandFees(txCtx)
		return oper, ops, err
	}
}

func SimulateMsgSubmitEthereumTxConfirmation(simCtx *simulateContext) simtypes.Operation {
	return simulateMsgSubmitEthereumTxConfirmation(simCtx)
}

func SimulateMsgSubmitEthereumEvent(simCtx *simulateContext) simtypes.Operation {
	return simulateMsgSubmitEthereumEvent(simCtx)
}

func SimulateMsgSendToEthereum(simCtx *simulateContext) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		ak := simCtx.ak
		bk := simCtx.bk
		acc, _ := simtypes.RandomAcc(r, accs)
		expendable := bk.SpendableCoin(ctx, acc.Address, sdk.DefaultBondDenom)
		amount := sdk.NewCoin(sdk.DefaultBondDenom, simtypes.RandomAmount(r, expendable.Amount))
		fee := sdk.NewCoin(sdk.DefaultBondDenom, simtypes.RandomAmount(r, expendable.Amount.Sub(amount.Amount)))

		msg := &types.MsgSendToEthereum{
			Sender:            acc.Address.String(),
			EthereumRecipient: randomEthAddress(r).String(),
			Amount:            amount,
			BridgeFee:         fee,
		}

		txCtx := simulation.OperationInput{
			R:               r,
			App:             app,
			TxGen:           simappparams.MakeTestEncodingConfig().TxConfig,
			Cdc:             nil,
			Msg:             msg,
			MsgType:         msg.Type(),
			Context:         ctx,
			SimAccount:      acc,
			AccountKeeper:   ak,
			Bankkeeper:      bk,
			ModuleName:      types.ModuleName,
			CoinsSpentInMsg: sdk.Coins{amount.Add(fee)},
		}

		oper, ops, err := simulation.GenAndDeliverTxWithRandFees(txCtx)

		whenCall := ctx.BlockHeader().Time.Add(time.Duration(r.Intn(maxWaitSeconds)+1) * time.Second)
		ops = append(ops, simtypes.FutureOperation{
			BlockTime: whenCall,
			Op:        simulateMsgRequestBatchTx(simCtx),
		})
		return oper, ops, err
	}
}

func SimulateMsgRequestBatchTx(simCtx *simulateContext) simtypes.Operation {
	return simulateMsgRequestBatchTx(simCtx)
}

func SimulateMsgCancelSendToEthereum(simCtx *simulateContext) simtypes.Operation {
	return simulateMsgCancelSendToEthereum(simCtx)
}

func SimulateMsgEthereumHeightVote(simCtx *simulateContext) simtypes.Operation {
	return simulateMsgEthereumHeightVote(simCtx)
}

func simulateMsgSubmitEthereumTxConfirmation(simCtx *simulateContext, txType string) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		gk := simCtx.gk

		acc, err := randomValidator(ctx, r, gk.StakingKeeper, accs)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, TypeMsgSubmitEthereumTxConfirmation, "no validator found"), nil, err
		}
		ethPrivKey, err := crypto.ToECDSA(acc.PrivKey.Bytes())
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, TypeMsgSubmitEthereumTxConfirmation, "can not get eth private key"), nil, err
		}
		ethAddr := common.BytesToAddress(acc.Address.Bytes()).String()
		orchAddr := acc.Address.String()

		var msg *types.MsgSubmitEthereumTxConfirmation

		switch txType {
		case "SignerSetTx":
			tx := gk.GetLatestSignerSetTx(ctx)

			if tx == nil {
				return simtypes.NoOpMsg(types.ModuleName, TypeMsgSubmitEthereumTxConfirmation, ""), nil, nil
			}

			gravityId := gk.GetParams(ctx).GravityId
			checkpoint := tx.GetCheckpoint([]byte(gravityId))
			signature, err := types.NewEthereumSignature(checkpoint, ethPrivKey)
			if err != nil {
				return simtypes.NoOpMsg(types.ModuleName, TypeMsgSubmitEthereumTxConfirmation, "can not sign the msg"), nil, err
			}

			signerSetTxConfirmation := &types.SignerSetTxConfirmation{
				SignerSetNonce: tx.Nonce,
				EthereumSigner: ethAddr,
				Signature:      signature,
			}

			confirmation, err := types.PackConfirmation(signerSetTxConfirmation)
			if err != nil {
				return simtypes.NoOpMsg(types.ModuleName, TypeMsgSubmitEthereumTxConfirmation, "failed to pack the confirmation"), nil, err
			}

			msg = &types.MsgSubmitEthereumTxConfirmation{
				Confirmation: confirmation,
				Signer:       orchAddr,
			}

		}

	}
}

func simulateMsgSubmitEthereumEvent(simCtx *simulateContext) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		return simtypes.OperationMsg{}, nil, nil
	}
}

func simulateMsgRequestBatchTx(simCtx *simulateContext) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		return simtypes.OperationMsg{}, nil, nil
	}
}

func simulateMsgCancelSendToEthereum(simCtx *simulateContext) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		return simtypes.OperationMsg{}, nil, nil
	}
}

func simulateMsgEthereumHeightVote(simCtx *simulateContext) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		return simtypes.OperationMsg{}, nil, nil
	}
}

func randomEthAddress(r *rand.Rand) common.Address {
	addr := make([]byte, 32)
	_, err := r.Read(addr)
	if err != nil {
		panic(err)
	}
	return common.BytesToAddress(addr)
}

func randomValidator(ctx sdk.Context, r *rand.Rand, sk types.StakingKeeper, accs []simtypes.Account) (simtypes.Account, error) {
	var valset []simtypes.Account
	for _, acc := range accs {
		val := sdk.ValAddress(acc.Address)
		if sk.Validator(ctx, val) != nil {
			valset = append(valset, acc)
		}
	}
	if len(valset) == 0 {
		return simtypes.Account{}, fmt.Errorf("none of the sim account is a validator")
	}
	val, _ := simtypes.RandomAcc(r, valset)
	return val, nil
}