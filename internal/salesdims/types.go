package salesdims

import "time"

const (
	ConflictSelectionRule = "most frequent variant; tie -> latest seen; tie -> lexical"
	GroupCodePolicy       = "group_code source unresolved in phase 2C-5.2; write NULL"
)

type SyncMode string

const (
	SyncModePlan  SyncMode = "plan"
	SyncModeApply SyncMode = "apply"
)

type SyncRequest struct {
	OwnerUserKey string
	OwnerUserID  int64
	StartDate    time.Time
	EndDate      time.Time
	Mode         SyncMode
}

type ProductCandidate struct {
	OwnerUserID    int64
	ProductNo      string
	ProductName    string
	CateNo         string
	CateName       string
	LastSeenAt     time.Time
	SourceRowCount int64
}

type BranchCandidate struct {
	OwnerUserID    int64
	BranchID       string
	BranchName     string
	GroupCode      string
	LastSeenAt     time.Time
	SourceRowCount int64
}

type ProductConflictSample struct {
	ProductNo            string
	VariantCount         int64
	ChosenProductName    string
	ChosenCateNo         string
	ChosenCateName       string
	ChosenSourceRowCount int64
	ChosenLastSeenAt     string
	SampleVariants       string
}

type BranchConflictSample struct {
	BranchID             string
	VariantCount         int64
	ChosenBranchName     string
	ChosenGroupCode      string
	ChosenSourceRowCount int64
	ChosenLastSeenAt     string
	SampleVariants       string
}

type PlanResult struct {
	Request                SyncRequest
	ProductCandidates      []ProductCandidate
	BranchCandidates       []BranchCandidate
	ProductConflictCount   int64
	BranchConflictCount    int64
	ProductConflictSamples []ProductConflictSample
	BranchConflictSamples  []BranchConflictSample
}

type ApplyResult struct {
	Plan               PlanResult
	ProductUpsertCount int64
	BranchUpsertCount  int64
	WrittenTables      []string
	SalesFactWritten   bool
}
