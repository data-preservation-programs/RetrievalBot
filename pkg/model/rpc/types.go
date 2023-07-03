package rpc

type Deal struct {
	Proposal DealProposal
	State    DealState
}

type Cid struct {
	Root string `json:"/" mapstructure:"/"`
}

type DealProposal struct {
	PieceCID     Cid
	PieceSize    uint64
	VerifiedDeal bool
	Client       string
	Provider     string
	Label        string
	StartEpoch   int32
	EndEpoch     int32
}

type DealState struct {
	SectorStartEpoch int32
	LastUpdatedEpoch int32
	SlashEpoch       int32
}
