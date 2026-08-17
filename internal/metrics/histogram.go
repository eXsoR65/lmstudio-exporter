package metrics

import "sort"

type histogram struct {
	buckets []float64
	counts  []uint64
	count   uint64
	sum     float64
}

func newHistogram(buckets []float64) *histogram {
	cpy := append([]float64(nil), buckets...)
	sort.Float64s(cpy)
	return &histogram{buckets: cpy, counts: make([]uint64, len(cpy))}
}

func (h *histogram) Observe(v float64) {
	h.count++
	h.sum += v
	for i, upper := range h.buckets {
		if v <= upper {
			h.counts[i]++
		}
	}
}
