package algo

type Progress struct {
	SegmentsDone  int
	TotalSegments int
	PrimesFound   uint64
	CurrentLow    uint64
	CurrentHigh   uint64
	End           uint64
}

type ProgressFunc func(Progress)
