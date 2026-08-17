package lms

type DaemonStatus struct {
	Running bool
	PID     int
}

type Model struct {
	Identifier    string
	Type          string
	Generating    bool
	QueueDepth    float64
	ContextLength float64
	Parallel      float64
	SizeBytes     float64
	Architecture  string
}

type PredictionStats struct {
	Model               string
	InputTokens         float64
	OutputTokens        float64
	ReasoningTokens     float64
	AcceptedDraftTokens float64
	TokensPerSecond     float64
	TimeToFirstToken    float64
	ModelLoadTime       float64
	GenerationTime      float64
	HasInputTokens      bool
	HasOutputTokens     bool
	HasReasoningTokens  bool
	HasDraftTokens      bool
	HasTokensPerSecond  bool
	HasTTFT             bool
	HasModelLoadTime    bool
	HasGenerationTime   bool
}
