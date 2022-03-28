package types

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// ProposalTypeCommunityPoolEthereumSpend defines the type for a CommunityPoolEthereumSpendProposal
	ProposalTypeCommunityPoolEthereumSpend = "CommunityPoolEthereumSpend"
)

// Assert CommunityPoolEthereumSpendProposal implements govtypes.Content at compile-time
var _ govtypes.Content = &CommunityPoolEthereumSpendProposal{}

func init() {
	govtypes.RegisterProposalType(ProposalTypeCommunityPoolEthereumSpend)
	govtypes.RegisterProposalTypeCodec(&CommunityPoolEthereumSpendProposal{}, "gravity/CommunityPoolEthereumSpendProposal")
}

// NewCommunityPoolEthereumSpendProposal creates a new community pool spend proposal.
//nolint:interfacer
func NewCommunityPoolEthereumSpendProposal(title, description string, recipient sdk.AccAddress, amount sdk.Coins) *CommunityPoolEthereumSpendProposal {
	return &CommunityPoolEthereumSpendProposal{title, description, recipient.String(), amount}
}

// GetTitle returns the title of a community pool Ethereum spend proposal.
func (csp *CommunityPoolEthereumSpendProposal) GetTitle() string { return csp.Title }

// GetDescription returns the description of a community pool Ethereum spend proposal.
func (csp *CommunityPoolEthereumSpendProposal) GetDescription() string { return csp.Description }

// GetDescription returns the routing key of a community pool Ethereum spend proposal.
func (csp *CommunityPoolEthereumSpendProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns the type of a community pool Ethereum spend proposal.
func (csp *CommunityPoolEthereumSpendProposal) ProposalType() string {
	return ProposalTypeCommunityPoolEthereumSpend
}

// ValidateBasic runs basic stateless validity checks
func (csp *CommunityPoolEthereumSpendProposal) ValidateBasic() error {
	err := govtypes.ValidateAbstract(csp)
	if err != nil {
		return err
	}
	if !csp.Amount.IsValid() {
		return ErrInvalidProposalAmount
	}
	if !common.IsHexAddress(csp.Recipient) {
		return ErrInvalidProposalRecipient
	}

	return nil
}

// String implements the Stringer interface.
func (csp CommunityPoolEthereumSpendProposal) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`Community Pool Spend Proposal:
  Title:       %s
  Description: %s
  Recipient:   %s
  Amount:      %s
`, csp.Title, csp.Description, csp.Recipient, csp.Amount))
	return b.String()
}
