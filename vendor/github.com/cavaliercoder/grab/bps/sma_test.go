package bps

import (
	"testing"
	"time"
)

type Sample struct {
	N      int64
	Expect float64
}

func getSimpleSamples(sampleCount, rate int) []Sample {
	a := make([]Sample, sampleCount)
	for i := 1; i < sampleCount; i++ {
		a[i] = Sample{N: int64(i * rate), Expect: float64(rate)}
	}
	return a
}

type SampleSetTest struct {
	Gauge    Gauge
	Interval time.Duration
	Samples  []Sample
}

func (c *SampleSetTest) Run(t *testing.T) {
	ts := time.Unix(0, 0)
	for i, sample := range c.Samples {
		c.Gauge.Sample(ts, sample.N)
		if actual := c.Gauge.BPS(); actual != sample.Expect {
			t.Errorf("expected: Gauge.BPS() → %0.2f, got %0.2f in test %d", sample.Expect, actual, i+1)
		}
		ts = ts.Add(c.Interval)
	}
}

func TestSMA_SimpleSteadyCase(t *testing.T) {
	test := &SampleSetTest{
		Interval: time.Second,
		Samples:  getSimpleSamples(100000, 3),
	}
	t.Run("SmallSampleSize", func(t *testing.T) {
		test.Gauge = NewSMA(2)
		test.Run(t)
	})
	t.Run("RegularSize", func(t *testing.T) {
		test.Gauge = NewSMA(6)
		test.Run(t)
	})
	t.Run("LargeSampleSize", func(t *testing.T) {
		test.Gauge = NewSMA(1000)
		test.Run(t)
	})
}
