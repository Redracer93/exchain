package types

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sdk "github.com/okex/exchain/libs/cosmos-sdk/types"
)

// Proposal defines a struct used by the governance module to allow for voting
// on network changes.
type Proposal struct {
	Content `json:"content" yaml:"content"` // Proposal content interface

	ProposalID       uint64         `json:"id" yaml:"id"`                                 //  ID of the proposal
	Status           ProposalStatus `json:"proposal_status" yaml:"proposal_status"`       // Status of the Proposal {Pending, Active, Passed, Rejected}
	FinalTallyResult TallyResult    `json:"final_tally_result" yaml:"final_tally_result"` // Result of Tallys

	SubmitTime     time.Time    `json:"submit_time" yaml:"submit_time"`           // Time of the block where TxGovSubmitProposal was included
	DepositEndTime time.Time    `json:"deposit_end_time" yaml:"deposit_end_time"` // Time that the Proposal would expire if deposit amount isn't met
	TotalDeposit   sdk.SysCoins `json:"total_deposit" yaml:"total_deposit"`       // Current deposit on this proposal. Initial value is set at InitialDeposit

	VotingStartTime time.Time `json:"voting_start_time" yaml:"voting_start_time"` // Time of the block where MinDeposit was reached. -1 if MinDeposit is not reached
	VotingEndTime   time.Time `json:"voting_end_time" yaml:"voting_end_time"`     // Time that the VotingPeriod for this proposal will end and votes will be tallied
}

func NewProposal(ctx sdk.Context, totalVoting sdk.Dec, content Content, id uint64, submitTime, depositEndTime time.Time) Proposal {
	return Proposal{
		Content:          content,
		ProposalID:       id,
		Status:           StatusDepositPeriod,
		FinalTallyResult: EmptyTallyResult(totalVoting),
		TotalDeposit:     sdk.SysCoins{},
		SubmitTime:       submitTime,
		DepositEndTime:   depositEndTime,
	}
}

// nolint
func (p Proposal) String() string {
	return fmt.Sprintf(`Proposal %d:
  Title:              %s
  Type:               %s
  Status:             %s
  Submit Time:        %s
  Deposit End Time:   %s
  Total Deposit:      %s
  Voting Start Time:  %s
  Voting End Time:    %s
  Description:        %s`,
		p.ProposalID, p.GetTitle(), p.ProposalType(),
		p.Status, p.SubmitTime, p.DepositEndTime,
		p.TotalDeposit, p.VotingStartTime, p.VotingEndTime, p.GetDescription(),
	)
}

// Proposals is an array of proposal
type Proposals []Proposal

// nolint
func (p Proposals) String() string {
	out := "ID - (Status) [Type] Title\n"
	for _, prop := range p {
		out += fmt.Sprintf("%d - (%s) [%s] %s\n",
			prop.ProposalID, prop.Status,
			prop.ProposalType(), prop.GetTitle())
	}
	return strings.TrimSpace(out)
}

// WrapProposalForCosmosAPI is for compatibility with the standard cosmos REST API
func WrapProposalForCosmosAPI(proposal Proposal, content Content) Proposal {
	return Proposal{
		Content:          content,
		ProposalID:       proposal.ProposalID,
		Status:           proposal.Status,
		FinalTallyResult: proposal.FinalTallyResult,
		SubmitTime:       proposal.SubmitTime,
		DepositEndTime:   proposal.DepositEndTime,
		TotalDeposit:     proposal.TotalDeposit,
		VotingStartTime:  proposal.VotingStartTime,
		VotingEndTime:    proposal.VotingEndTime,
	}
}

type (
	// ProposalQueue
	ProposalQueue []uint64

	// ProposalStatus is a type alias that represents a proposal status as a byte
	ProposalStatus byte
)

//nolint
const (
	StatusNil           ProposalStatus = 0x00
	StatusDepositPeriod ProposalStatus = 0x01
	StatusVotingPeriod  ProposalStatus = 0x02
	StatusPassed        ProposalStatus = 0x03
	StatusRejected      ProposalStatus = 0x04
	StatusFailed        ProposalStatus = 0x05
)

// ProposalStatusToString turns a string into a ProposalStatus
func ProposalStatusFromString(str string) (ProposalStatus, error) {
	switch str {
	case "DepositPeriod":
		return StatusDepositPeriod, nil

	case "VotingPeriod":
		return StatusVotingPeriod, nil

	case "Passed":
		return StatusPassed, nil

	case "Rejected":
		return StatusRejected, nil

	case "Failed":
		return StatusFailed, nil

	case "":
		return StatusNil, nil

	default:
		return ProposalStatus(0xff), fmt.Errorf("'%s' is not a valid proposal status", str)
	}
}

// ValidProposalStatus returns true if the proposal status is valid and false
// otherwise.
func ValidProposalStatus(status ProposalStatus) bool {
	if status == StatusDepositPeriod ||
		status == StatusVotingPeriod ||
		status == StatusPassed ||
		status == StatusRejected ||
		status == StatusFailed {
		return true
	}
	return false
}

// Marshal needed for protobuf compatibility
func (status ProposalStatus) Marshal() ([]byte, error) {
	return []byte{byte(status)}, nil
}

// Unmarshal needed for protobuf compatibility
func (status *ProposalStatus) Unmarshal(data []byte) error {
	*status = ProposalStatus(data[0])
	return nil
}

// Marshals to JSON using string
func (status ProposalStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(status.String())
}

// Unmarshals from JSON assuming Bech32 encoding
func (status *ProposalStatus) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}

	bz2, err := ProposalStatusFromString(s)
	if err != nil {
		return err
	}

	*status = bz2
	return nil
}

// String implements the Stringer interface.
func (status ProposalStatus) String() string {
	switch status {
	case StatusDepositPeriod:
		return "DepositPeriod"

	case StatusVotingPeriod:
		return "VotingPeriod"

	case StatusPassed:
		return "Passed"

	case StatusRejected:
		return "Rejected"

	case StatusFailed:
		return "Failed"

	default:
		return ""
	}
}

func (status ProposalStatus) MarshalYAML() (interface{}, error) {
	switch status {
	case StatusDepositPeriod:
		return "DepositPeriod", nil

	case StatusVotingPeriod:
		return "VotingPeriod", nil

	case StatusPassed:
		return "Passed", nil

	case StatusRejected:
		return "Rejected", nil

	case StatusFailed:
		return "Failed", nil

	default:
		return "", nil
	}
}

// Format implements the fmt.Formatter interface.
// nolint: errcheck
func (status ProposalStatus) Format(s fmt.State, verb rune) {
	switch verb {
	case 's':
		s.Write([]byte(status.String()))
	default:
		// TODO: Do this conversion more directly
		s.Write([]byte(fmt.Sprintf("%v", byte(status))))
	}
}

// Tally Results
type TallyResult struct {
	// total power of accounts whose votes are voted to the current validator set
	TotalPower sdk.Dec `json:"total_power"`
	// total power of accounts who has voted for a proposal
	TotalVotedPower sdk.Dec `json:"total_voted_power"`
	Yes             sdk.Dec `json:"yes"`
	Abstain         sdk.Dec `json:"abstain"`
	No              sdk.Dec `json:"no"`
	NoWithVeto      sdk.Dec `json:"no_with_veto"`
}

func NewTallyResult(yes, abstain, no, noWithVeto sdk.Dec) TallyResult {
	return TallyResult{
		Yes:        yes,
		Abstain:    abstain,
		No:         no,
		NoWithVeto: noWithVeto,
	}
}

func NewTallyResultFromMap(results map[VoteOption]sdk.Dec) TallyResult {
	return TallyResult{
		Yes:        results[OptionYes],
		Abstain:    results[OptionAbstain],
		No:         results[OptionNo],
		NoWithVeto: results[OptionNoWithVeto],
	}
}

// EmptyTallyResult returns an empty TallyResult.
func EmptyTallyResult(totalVoting sdk.Dec) TallyResult {
	return TallyResult{
		TotalPower:      totalVoting,
		TotalVotedPower: sdk.ZeroDec(),
		Yes:             sdk.ZeroDec(),
		Abstain:         sdk.ZeroDec(),
		No:              sdk.ZeroDec(),
		NoWithVeto:      sdk.ZeroDec(),
	}
}

// Equals returns if two proposals are equal.
func (tr TallyResult) Equals(comp TallyResult) bool {
	return tr.Yes.Equal(comp.Yes) &&
		tr.Abstain.Equal(comp.Abstain) &&
		tr.No.Equal(comp.No) &&
		tr.NoWithVeto.Equal(comp.NoWithVeto)
}

func (tr TallyResult) String() string {
	return fmt.Sprintf(`Tally Result:
  TotalPower %s
  TotalVotedPower %s
  Yes:        %s
  Abstain:    %s
  No:         %s
  NoWithVeto: %s`, tr.TotalPower, tr.TotalVotedPower, tr.Yes, tr.Abstain, tr.No, tr.NoWithVeto)
}

// Proposal types
const (
	ProposalTypeText            string = "Text"
	ProposalTypeSoftwareUpgrade string = "SoftwareUpgrade"
)

// Text Proposal
type TextProposal struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
}

func NewTextProposal(title, description string) Content {
	return TextProposal{title, description}
}

// Implements Proposal Interface
var _ Content = TextProposal{}

// nolint
func (tp TextProposal) GetTitle() string         { return tp.Title }
func (tp TextProposal) GetDescription() string   { return tp.Description }
func (tp TextProposal) ProposalRoute() string    { return RouterKey }
func (tp TextProposal) ProposalType() string     { return ProposalTypeText }
func (tp TextProposal) ValidateBasic() sdk.Error { return ValidateAbstract(DefaultCodespace, tp) }

func (tp TextProposal) String() string {
	return fmt.Sprintf(`Text Proposal:
  Title:       %s
  Description: %s
`, tp.Title, tp.Description)
}

// Software Upgrade Proposals
// TODO: We have to add fields for SUP specific arguments e.g. commit hash,
// upgrade date, etc.
type SoftwareUpgradeProposal struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
}

func NewSoftwareUpgradeProposal(title, description string) Content {
	return SoftwareUpgradeProposal{title, description}
}

// Implements Proposal Interface
var _ Content = SoftwareUpgradeProposal{}

// nolint
func (sup SoftwareUpgradeProposal) GetTitle() string       { return sup.Title }
func (sup SoftwareUpgradeProposal) GetDescription() string { return sup.Description }
func (sup SoftwareUpgradeProposal) ProposalRoute() string  { return RouterKey }
func (sup SoftwareUpgradeProposal) ProposalType() string   { return ProposalTypeSoftwareUpgrade }
func (sup SoftwareUpgradeProposal) ValidateBasic() sdk.Error {
	return ValidateAbstract(DefaultCodespace, sup)
}

func (sup SoftwareUpgradeProposal) String() string {
	return fmt.Sprintf(`Software Upgrade Proposal:
  Title:       %s
  Description: %s
`, sup.Title, sup.Description)
}

var validProposalTypes = map[string]struct{}{
	ProposalTypeText:            {},
	ProposalTypeSoftwareUpgrade: {},
}

// RegisterProposalType registers a proposal type. It will panic if the type is
// already registered.
func RegisterProposalType(ty string) {
	if _, ok := validProposalTypes[ty]; ok {
		panic(fmt.Sprintf("already registered proposal type: %s", ty))
	}

	validProposalTypes[ty] = struct{}{}
}

// ContentFromProposalType returns a Content object based on the proposal type.
func ContentFromProposalType(title, desc, ty string) Content {
	switch ty {
	case ProposalTypeText:
		return NewTextProposal(title, desc)

	case ProposalTypeSoftwareUpgrade:
		return NewSoftwareUpgradeProposal(title, desc)

	default:
		return nil
	}
}

// IsValidProposalType returns a boolean determining if the proposal type is
// valid.
//
// NOTE: Modules with their own proposal types must register them.
func IsValidProposalType(ty string) bool {
	_, ok := validProposalTypes[ty]
	return ok
}

// ProposalHandler implements the Handler interface for governance module-based
// proposals (ie. TextProposal and SoftwareUpgradeProposal). Since these are
// merely signaling mechanisms at the moment and do not affect state, it
// performs a no-op.
func ProposalHandler(_ sdk.Context, p *Proposal) sdk.Error {
	switch p.ProposalType() {
	case ProposalTypeText, ProposalTypeSoftwareUpgrade:
		// both proposal types do not change state so this performs a no-op
		return nil

	default:
		errMsg := fmt.Sprintf("unrecognized gov proposal type: %s", p.ProposalType())
		return sdk.ErrUnknownRequest(errMsg)
	}
}
