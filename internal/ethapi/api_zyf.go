package ethapi

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type Call2Resp struct {
	UsedGas      uint64
	CallResult   hexutil.Bytes
	BalanceDelta *big.Int
}

func (s *BlockChainAPI) Call2(ctx context.Context, args TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash, overrides *StateOverride) (hexutil.Bytes, error) {
	result, err, balanceDelta := DoCall2(ctx, s.b, args, blockNrOrHash, overrides, s.b.RPCEVMTimeout(), s.b.RPCGasCap(), nil)
	if err != nil {
		return nil, err
	}
	// If the result contains a revert reason, try to unpack and return it.
	if len(result.Revert()) > 0 {
		return nil, newRevertError(result)
	}
	resp := &Call2Resp{
		UsedGas:      result.UsedGas,
		CallResult:   result.Return(),
		BalanceDelta: balanceDelta,
	}
	bs, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("marshal failed: %v, data: %v", err, resp)
	}
	return bs, result.Err
}

func DoCall2(ctx context.Context, b Backend, args TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash, overrides *StateOverride, timeout time.Duration, globalGasCap uint64, simulateBalance *big.Int) (*core.ExecutionResult, error, *big.Int) {
	defer func(start time.Time) { log.Debug("Executing EVM call finished", "runtime", time.Since(start)) }(time.Now())

	state, header, err := b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err, nil
	}
	if err := overrides.Apply(state); err != nil {
		return nil, err, nil
	}
	// Setup context so it may be cancelled the call has completed
	// or, in case of unmetered gas, setup a context with a timeout.
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	// Make sure the context is cancelled when the call has completed
	// this makes sure resources are cleaned up.
	defer cancel()

	// Get a new instance of the EVM.
	msg, err := args.ToMessage(globalGasCap, header.BaseFee)
	if err != nil {
		return nil, err, nil
	}
	evm, vmError, err := b.GetEVM(ctx, msg, state, header, &vm.Config{NoBaseFee: true})
	if err != nil {
		return nil, err, nil
	}
	// Wait for the context to be done and cancel the evm. Even if the
	// EVM has finished, cancelling may be done (repeatedly)
	go func() {
		<-ctx.Done()
		evm.Cancel()
	}()

	if simulateBalance != nil {
		state.SetBalance(msg.From(), simulateBalance)
	}
	senderBalance := state.GetBalance(msg.From())
	// Execute the message.
	gp := new(core.GasPool).AddGas(math.MaxUint64)
	result, err := core.ApplyMessage(evm, msg, gp)
	if err := vmError(); err != nil {
		return nil, err, nil
	}
	balanceDelta := senderBalance.Sub(state.GetBalance(msg.From()), senderBalance)

	// If the timer caused an abort, return an appropriate error message
	if evm.Cancelled() {
		return nil, fmt.Errorf("execution aborted (timeout = %v)", timeout), nil
	}
	if err != nil {
		return result, fmt.Errorf("err: %w (supplied gas %d)", err, msg.Gas()), nil
	}
	return result, nil, balanceDelta
}

func (s *BlockChainAPI) Call3(ctx context.Context, args TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash, overrides *StateOverride) (hexutil.Bytes, error) {
	eth100, _ := new(big.Int).SetString("100000000000000000000", 10)
	result, err, balanceDelta := DoCall2(ctx, s.b, args, blockNrOrHash, overrides, s.b.RPCEVMTimeout(), s.b.RPCGasCap(), eth100)
	if err != nil {
		return nil, err
	}
	// If the result contains a revert reason, try to unpack and return it.
	if len(result.Revert()) > 0 {
		return nil, newRevertError(result)
	}
	resp := &Call2Resp{
		UsedGas:      result.UsedGas,
		CallResult:   result.Return(),
		BalanceDelta: balanceDelta,
	}
	bs, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("marshal failed: %v, data: %v", err, resp)
	}
	return bs, result.Err
}
