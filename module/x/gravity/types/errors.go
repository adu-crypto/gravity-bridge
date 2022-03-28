package types

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	ErrInvalid                  = sdkerrors.Register(ModuleName, 3, "invalid")
	ErrSupplyOverflow           = sdkerrors.Register(ModuleName, 4, "malicious ERC20 with invalid supply sent over bridge")
	ErrDelegateKeys             = sdkerrors.Register(ModuleName, 5, "failed to delegate keys")
	ErrEmptyEthSig              = sdkerrors.Register(ModuleName, 6, "empty Ethereum signature")
	ErrInvalidERC20Event        = sdkerrors.Register(ModuleName, 7, "invalid ERC20 deployed event")
	ErrInvalidProposalAmount    = sdkerrors.Register(ModuleName, 8, "invalid community pool Ethereum spend proposal amount")
	ErrInvalidProposalRecipient = sdkerrors.Register(ModuleName, 9, "invalid community pool Ethereum spend proposal recipient")
)
