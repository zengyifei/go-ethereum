package ethapi

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	ERC20ABI    = "[{\"constant\":true,\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_spender\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_from\",\"type\":\"address\"},{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"name\":\"\",\"type\":\"uint8\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"name\":\"balance\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_to\",\"type\":\"address\"},{\"name\":\"_value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_owner\",\"type\":\"address\"},{\"name\":\"_spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"fallback\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"}]"
	WETHABI     = `[{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"guy","type":"address"},{"name":"wad","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"src","type":"address"},{"name":"dst","type":"address"},{"name":"wad","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"wad","type":"uint256"}],"name":"withdraw","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"dst","type":"address"},{"name":"wad","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[],"name":"deposit","outputs":[],"payable":true,"stateMutability":"payable","type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"},{"name":"","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"payable":true,"stateMutability":"payable","type":"fallback"},{"anonymous":false,"inputs":[{"indexed":true,"name":"src","type":"address"},{"indexed":true,"name":"guy","type":"address"},{"indexed":false,"name":"wad","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"src","type":"address"},{"indexed":true,"name":"dst","type":"address"},{"indexed":false,"name":"wad","type":"uint256"}],"name":"Transfer","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"dst","type":"address"},{"indexed":false,"name":"wad","type":"uint256"}],"name":"Deposit","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"src","type":"address"},{"indexed":false,"name":"wad","type":"uint256"}],"name":"Withdrawal","type":"event"}]`
	erc20, _    = abi.JSON(strings.NewReader(ERC20ABI))
	weth, _     = abi.JSON(strings.NewReader(WETHABI))
	wethAddress = common.HexToAddress("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2")
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
		// state.SetBalance(msg.From(), new(big.Int).Mul(simulateBalance, big.NewInt(2)))
		// if err := depositWETH(evm, msg.From(), wethAddress, simulateBalance, header, globalGasCap); err != nil {
		// return nil, err, nil
		// }
	}
	senderBalance := state.GetBalance(msg.From())
	// wethBalance, err := getTokenBalance(evm, msg.From(), wethAddress, header, globalGasCap)
	if err != nil {
		fmt.Println("getWethBalance failed", err)
		return nil, err, nil
	}
	// fmt.Println(msg.From(), "wethBalance1", wethBalance)
	// Execute the message.
	gp := new(core.GasPool).AddGas(math.MaxUint64)
	result, err := core.ApplyMessage(evm, msg, gp)
	if err := vmError(); err != nil {
		return nil, err, nil
	}
	// wethBalance2, err := getTokenBalance(evm, msg.From(), wethAddress, header, globalGasCap)
	// if err != nil {
	// fmt.Println("getWethBalance2 failed", err)
	// return nil, err, nil
	// }
	// fmt.Println(msg.From(), "wethBalance2", wethBalance2)
	// ethDelta := new(big.Int).Sub(state.GetBalance(msg.From()), senderBalance)
	// wethDelta := new(big.Int).Sub(wethBalance2, wethBalance)
	// balanceDelta := new(big.Int).Add(ethDelta, wethDelta)
	balanceDelta := new(big.Int).Sub(state.GetBalance(msg.From()), senderBalance)

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

func getTokenBalance(evm *vm.EVM, owner, tokenAddress common.Address, header *types.Header, globalGasCap uint64) (*big.Int, error) {
	method := "balanceOf"
	input, err := erc20.Pack(method, owner)
	if err != nil {
		fmt.Println("getTokenBalance1", err)
		return *new(*big.Int), err
	}
	inputData := hexutil.Bytes(input)
	txArgs := TransactionArgs{
		To:   &tokenAddress,
		Data: &inputData,
	}
	msg, err := txArgs.ToMessage(globalGasCap, header.BaseFee)
	if err != nil {
		fmt.Println("getTokenBalance2", err)
		return *new(*big.Int), err
	}

	sender := vm.AccountRef(msg.From())
	output, _, vmerr := evm.Call(sender, *msg.To(), msg.Data(), msg.Gas(), msg.Value())
	if vmerr != nil {
		fmt.Println("getTokenBalance3", vmerr)
		return *new(*big.Int), vmerr
	}

	out, err := erc20.Unpack(method, output)
	if err != nil {
		fmt.Println("getTokenBalance4", err)
		return *new(*big.Int), err
	}

	bal := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	return bal, err
}

func depositWETH(evm *vm.EVM, from, tokenAddress common.Address, amount *big.Int, header *types.Header, globalGasCap uint64) error {
	method := "deposit"
	input, err := weth.Pack(method)
	if err != nil {
		fmt.Println("depositWETH1", err)
		return err
	}
	inputData := hexutil.Bytes(input)
	txArgs := TransactionArgs{
		From:  &from,
		To:    &tokenAddress,
		Data:  &inputData,
		Value: (*hexutil.Big)(amount),
	}
	msg, err := txArgs.ToMessage(globalGasCap, header.BaseFee)
	if err != nil {
		fmt.Println("depositWETH2", err)
		return err
	}

	sender := vm.AccountRef(msg.From())
	_, _, vmerr := evm.Call(sender, *msg.To(), msg.Data(), msg.Gas(), msg.Value())
	if vmerr != nil {
		fmt.Println("depositWETH3", err)
	}
	return vmerr
}
