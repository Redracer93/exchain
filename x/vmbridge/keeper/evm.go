package keeper

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	ethermint "github.com/okex/exchain/app/types"
	sdk "github.com/okex/exchain/libs/cosmos-sdk/types"
	sdkerrors "github.com/okex/exchain/libs/cosmos-sdk/types/errors"
	tmtypes "github.com/okex/exchain/libs/tendermint/types"
	erc20types "github.com/okex/exchain/x/erc20/types"
	evmtypes "github.com/okex/exchain/x/evm/types"
	"github.com/okex/exchain/x/vmbridge/types"
	"math/big"
)

// event __SendToWasmEventName(string wasmAddr,string recipient, string amount)
type SendToWasmEventHandler struct {
	Keeper
}

func NewSendToWasmEventHandler(k Keeper) *SendToWasmEventHandler {
	return &SendToWasmEventHandler{k}
}

// EventID Return the id of the log signature it handles
func (h SendToWasmEventHandler) EventID() common.Hash {
	return types.SendToWasmEvent.ID
}

// Handle Process the log
func (h SendToWasmEventHandler) Handle(ctx sdk.Context, contract common.Address, data []byte) error {
	if !tmtypes.HigherThanEarth(ctx.BlockHeight()) {
		errMsg := fmt.Sprintf("vmbridger not supprt at height %d", ctx.BlockHeight())
		return sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, errMsg)
	}

	params := h.wasmKeeper.GetParams(ctx)
	if !params.VmbridgeEnable {
		return types.ErrVMBridgeEnable
	}

	logger := h.Keeper.Logger()
	unpacked, err := types.SendToWasmEvent.Inputs.Unpack(data)
	if err != nil {
		// log and ignore
		logger.Error("log signature matches but failed to decode", "error", err)
		return nil
	}

	caller := sdk.AccAddress(contract.Bytes())
	wasmAddr := unpacked[0].(string)
	recipient := unpacked[1].(string)
	amount := sdk.NewIntFromBigInt(unpacked[2].(*big.Int))

	return h.Keeper.SendToWasm(ctx, caller, wasmAddr, recipient, amount)
}

// wasm call evm for erc20 exchange cw20,
func (k Keeper) SendToEvm(ctx sdk.Context, caller, contract string, recipient string, amount sdk.Int) (success bool, err error) {
	if !sdk.IsETHAddress(recipient) {
		return false, types.ErrIsNotETHAddr
	}

	if !sdk.IsETHAddress(contract) {
		return false, types.ErrIsNotETHAddr
	}

	contractAccAddr, err := sdk.AccAddressFromBech32(contract)
	if err != nil {
		return false, err
	}
	conrtractAddr := common.BytesToAddress(contractAccAddr.Bytes())

	recipientAccAddr, err := sdk.AccAddressFromBech32(recipient)
	if err != nil {
		return false, err
	}
	recipientAddr := common.BytesToAddress(recipientAccAddr.Bytes())
	input, err := types.GetMintERC20Input(caller, recipientAddr, amount.BigInt())
	if err != nil {
		return false, err
	}
	_, result, err := k.CallEvm(ctx, &conrtractAddr, big.NewInt(0), input)
	if err != nil {
		return false, err
	}
	return types.GetMintERC20Output(result.Ret)
}

// callEvm execute an evm message from native module
func (k Keeper) CallEvm(ctx sdk.Context, to *common.Address, value *big.Int, data []byte) (*evmtypes.ExecutionResult, *evmtypes.ResultData, error) {
	callerAddr := erc20types.IbcEvmModuleETHAddr

	config, found := k.evmKeeper.GetChainConfig(ctx)
	if !found {
		return nil, nil, types.ErrChainConfigNotFound
	}

	chainIDEpoch, err := ethermint.ParseChainID(ctx.ChainID())
	if err != nil {
		return nil, nil, err
	}

	acc := k.accountKeeper.GetAccount(ctx, callerAddr.Bytes())
	if acc == nil {
		acc = k.accountKeeper.NewAccountWithAddress(ctx, callerAddr.Bytes())
	}
	nonce := acc.GetSequence()
	txHash := tmtypes.Tx(ctx.TxBytes()).Hash(ctx.BlockHeight())
	ethTxHash := common.BytesToHash(txHash)

	gasLimit := ctx.GasMeter().Limit()
	if gasLimit == sdk.NewInfiniteGasMeter().Limit() {
		gasLimit = k.evmKeeper.GetParams(ctx).MaxGasLimitPerTx
	}

	st := evmtypes.StateTransition{
		AccountNonce: nonce,
		Price:        big.NewInt(0),
		GasLimit:     gasLimit,
		Recipient:    to,
		Amount:       value,
		Payload:      data,
		Csdb:         evmtypes.CreateEmptyCommitStateDB(k.evmKeeper.GenerateCSDBParams(), ctx),
		ChainID:      chainIDEpoch,
		TxHash:       &ethTxHash,
		Sender:       callerAddr,
		Simulate:     ctx.IsCheckTx(),
		TraceTx:      false,
		TraceTxLog:   false,
	}

	executionResult, resultData, err, _, _ := st.TransitionDb(ctx, config)
	if !ctx.IsCheckTx() && !ctx.IsTraceTx() {
		//TODO maybe add innertx
		//k.addEVMInnerTx(ethTxHash.Hex(), innertxs, contracts)
	}
	if err != nil {
		return nil, nil, err
	}

	st.Csdb.Commit(false) // write code to db
	acc.SetSequence(nonce + 1)
	k.accountKeeper.SetAccount(ctx, acc)

	return executionResult, resultData, err
}
