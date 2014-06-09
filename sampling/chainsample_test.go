package sampling

import "testing"

func TestChainSampler(t *testing.T) {
	numSamples := 10
	windowSize := 10 * numSamples
	numObservations := 10 * windowSize
	numSampleRounds := 10 * numObservations

	s := NewChainSampler(numSamples, windowSize)
	hist := make(map[int]int, numSamples)
	for i := 0; i < numSampleRounds; i++ {
		sampleStream(hist, numObservations, s)
	}
	ratio := histStddev(hist) / histMean(hist)
	if ratio > 1.05 {
		// XXX(dengnan): better sampler?
		t.Errorf("std dev: %v; mean: %v. Either we have a really bad PRNG, or a bad implementation", histStddev(hist), histMean(hist))
	}
	if len(hist) > windowSize {
		t.Errorf("sampled %v data. larger than window size %v", len(hist), windowSize)
	}
	for seqNum, freq := range hist {
		if seqNum < numObservations-windowSize && freq > 0 {
			t.Errorf("observation with seqnum %v is sampled %v times", seqNum, freq)
		}
	}
}
