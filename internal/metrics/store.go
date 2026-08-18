package metrics

import (
	"sort"
	"sync"
	"time"

	"github.com/eXsoR65/lmstudio-exporter/internal/lms"
)

type ModelState struct {
	Identifier    string
	Type          string
	Generating    bool
	QueueDepth    float64
	ContextLength float64
	Parallel      float64
	SizeBytes     float64
}

type InferenceState struct {
	RequestsTotal               uint64
	InputTokensTotal            float64
	OutputTokensTotal           float64
	ReasoningTokensTotal        float64
	DraftTokensTotal            float64
	LastInputTokens             float64
	HasLastInputTokens          bool
	LastOutputTokens            float64
	HasLastOutputTokens         bool
	LastContextTokens           float64
	HasLastContextTokens        bool
	LastContextUtilizationRatio float64
	HasLastContextUtilization   bool
	LastTokensPerSecond         float64
	HasLastTPS                  bool
	TPS                         *histogram
	TTFT                        *histogram
	ModelLoad                   *histogram
	Generation                  *histogram
}

type Store struct {
	mu sync.RWMutex

	Version   string
	Commit    string
	BuildDate string

	DaemonUp         bool
	DaemonPID        int
	PollSuccess      bool
	LastPollUnix     float64
	LogStreamEnabled bool
	LogStreamUp      bool
	LogEventsTotal   uint64
	LogParseErrors   uint64
	Models           map[string]ModelState
	Inference        map[string]*InferenceState
}

func NewStore(version, commit, buildDate string) *Store {
	return &Store{
		Version:   version,
		Commit:    commit,
		BuildDate: buildDate,
		Models:    make(map[string]ModelState),
		Inference: make(map[string]*InferenceState),
	}
}

func (s *Store) SetPoll(status lms.DaemonStatus, models []lms.Model, success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DaemonUp = status.Running
	s.DaemonPID = status.PID
	s.PollSuccess = success
	s.LastPollUnix = float64(time.Now().Unix())

	if !success {
		return
	}
	newModels := make(map[string]ModelState, len(models))
	for _, m := range models {
		newModels[m.Identifier] = ModelState{
			Identifier:    m.Identifier,
			Type:          m.Type,
			Generating:    m.Generating,
			QueueDepth:    m.QueueDepth,
			ContextLength: m.ContextLength,
			Parallel:      m.Parallel,
			SizeBytes:     m.SizeBytes,
		}
	}
	s.Models = newModels
}

func (s *Store) SetDaemonOnly(status lms.DaemonStatus, success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.DaemonUp = status.Running
	s.DaemonPID = status.PID
	s.PollSuccess = success
	s.LastPollUnix = float64(time.Now().Unix())
	if success && !status.Running {
		s.Models = make(map[string]ModelState)
	}
}

func (s *Store) SetLogStreamEnabled(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LogStreamEnabled = enabled
}

func (s *Store) SetLogStreamUp(up bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LogStreamUp = up
}

func (s *Store) IncLogEvent() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LogEventsTotal++
}

func (s *Store) IncLogParseError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LogParseErrors++
}

func (s *Store) ObservePrediction(stats lms.PredictionStats) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := s.Inference[stats.Model]
	if state == nil {
		state = &InferenceState{
			TPS:        newHistogram([]float64{1, 2, 5, 10, 20, 30, 40, 60, 100, 200}),
			TTFT:       newHistogram([]float64{0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60}),
			ModelLoad:  newHistogram([]float64{0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60, 120}),
			Generation: newHistogram([]float64{0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60, 120, 300, 600}),
		}
		s.Inference[stats.Model] = state
	}
	state.RequestsTotal++
	if stats.HasInputTokens {
		state.InputTokensTotal += stats.InputTokens
		state.LastInputTokens = stats.InputTokens
		state.HasLastInputTokens = true

		contextTokens := stats.InputTokens
		if stats.HasOutputTokens {
			contextTokens += stats.OutputTokens
		}
		if model, ok := s.Models[stats.Model]; ok && model.ContextLength > 0 {
			state.LastContextTokens = contextTokens
			state.HasLastContextTokens = true
			state.LastContextUtilizationRatio = contextTokens / model.ContextLength
			state.HasLastContextUtilization = true
		}
	}
	if stats.HasOutputTokens {
		state.OutputTokensTotal += stats.OutputTokens
		state.LastOutputTokens = stats.OutputTokens
		state.HasLastOutputTokens = true
	}
	if stats.HasReasoningTokens {
		state.ReasoningTokensTotal += stats.ReasoningTokens
	}
	if stats.HasDraftTokens {
		state.DraftTokensTotal += stats.AcceptedDraftTokens
	}
	if stats.HasTokensPerSecond {
		state.LastTokensPerSecond = stats.TokensPerSecond
		state.HasLastTPS = true
		state.TPS.Observe(stats.TokensPerSecond)
	}
	if stats.HasTTFT {
		state.TTFT.Observe(stats.TimeToFirstToken)
	}
	if stats.HasModelLoadTime {
		state.ModelLoad.Observe(stats.ModelLoadTime)
	}
	if stats.HasGenerationTime {
		state.Generation.Observe(stats.GenerationTime)
	}
}

func (s *Store) Ready() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.PollSuccess && s.DaemonUp
}

type Snapshot struct {
	Version, Commit, BuildDate string
	DaemonUp                   bool
	DaemonPID                  int
	PollSuccess                bool
	LastPollUnix               float64
	LogStreamEnabled           bool
	LogStreamUp                bool
	LogEventsTotal             uint64
	LogParseErrors             uint64
	Models                     []ModelState
	Inference                  map[string]InferenceSnapshot
}

type HistogramSnapshot struct {
	Buckets []float64
	Counts  []uint64
	Count   uint64
	Sum     float64
}

type InferenceSnapshot struct {
	RequestsTotal                    uint64
	InputTokensTotal                 float64
	OutputTokensTotal                float64
	ReasoningTokensTotal             float64
	DraftTokensTotal                 float64
	LastInputTokens                  float64
	HasLastInputTokens               bool
	LastOutputTokens                 float64
	HasLastOutputTokens              bool
	LastContextTokens                float64
	HasLastContextTokens             bool
	LastContextUtilizationRatio      float64
	HasLastContextUtilization        bool
	LastTokensPerSecond              float64
	HasLastTPS                       bool
	TPS, TTFT, ModelLoad, Generation HistogramSnapshot
}

func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snap := Snapshot{
		Version:          s.Version,
		Commit:           s.Commit,
		BuildDate:        s.BuildDate,
		DaemonUp:         s.DaemonUp,
		DaemonPID:        s.DaemonPID,
		PollSuccess:      s.PollSuccess,
		LastPollUnix:     s.LastPollUnix,
		LogStreamEnabled: s.LogStreamEnabled,
		LogStreamUp:      s.LogStreamUp,
		LogEventsTotal:   s.LogEventsTotal,
		LogParseErrors:   s.LogParseErrors,
		Inference:        make(map[string]InferenceSnapshot, len(s.Inference)),
	}
	for _, model := range s.Models {
		snap.Models = append(snap.Models, model)
	}
	sort.Slice(snap.Models, func(i, j int) bool { return snap.Models[i].Identifier < snap.Models[j].Identifier })
	for model, inf := range s.Inference {
		snap.Inference[model] = InferenceSnapshot{
			RequestsTotal:               inf.RequestsTotal,
			InputTokensTotal:            inf.InputTokensTotal,
			OutputTokensTotal:           inf.OutputTokensTotal,
			ReasoningTokensTotal:        inf.ReasoningTokensTotal,
			DraftTokensTotal:            inf.DraftTokensTotal,
			LastInputTokens:             inf.LastInputTokens,
			HasLastInputTokens:          inf.HasLastInputTokens,
			LastOutputTokens:            inf.LastOutputTokens,
			HasLastOutputTokens:         inf.HasLastOutputTokens,
			LastContextTokens:           inf.LastContextTokens,
			HasLastContextTokens:        inf.HasLastContextTokens,
			LastContextUtilizationRatio: inf.LastContextUtilizationRatio,
			HasLastContextUtilization:   inf.HasLastContextUtilization,
			LastTokensPerSecond:         inf.LastTokensPerSecond,
			HasLastTPS:                  inf.HasLastTPS,
			TPS:                         snapshotHistogram(inf.TPS),
			TTFT:                        snapshotHistogram(inf.TTFT),
			ModelLoad:                   snapshotHistogram(inf.ModelLoad),
			Generation:                  snapshotHistogram(inf.Generation),
		}
	}
	return snap
}

func snapshotHistogram(h *histogram) HistogramSnapshot {
	if h == nil {
		return HistogramSnapshot{}
	}
	return HistogramSnapshot{
		Buckets: append([]float64(nil), h.buckets...),
		Counts:  append([]uint64(nil), h.counts...),
		Count:   h.count,
		Sum:     h.sum,
	}
}
