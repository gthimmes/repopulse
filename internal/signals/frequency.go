package signals

import (
	"math"
	"time"

	"mood-ring/internal/types"
)

// ComputeFrequency computes the commit frequency signal.
// Scoring: coefficient-of-variation (60%) + longest-gap share (40%).
func ComputeFrequency(commits []types.CommitRecord, windowDays int) types.FrequencySignal {
	if windowDays < 1 {
		windowDays = 1
	}

	// Build daily buckets for the whole window, anchored to end-of-today.
	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999000000, now.Location())

	bucketOrder := make([]string, 0, windowDays)
	bucketCount := map[string]int{}
	for i := 0; i < windowDays; i++ {
		d := now.AddDate(0, 0, -(windowDays - 1 - i))
		key := d.UTC().Format("2006-01-02")
		bucketOrder = append(bucketOrder, key)
		bucketCount[key] = 0
	}

	for _, c := range commits {
		key := c.Date.UTC().Format("2006-01-02")
		if _, ok := bucketCount[key]; ok {
			bucketCount[key]++
		}
	}

	dailyBuckets := make([]types.DayBucket, len(bucketOrder))
	for i, k := range bucketOrder {
		dailyBuckets[i] = types.DayBucket{Date: k, Count: bucketCount[k]}
	}
	counts := make([]float64, len(dailyBuckets))
	for i, b := range dailyBuckets {
		counts[i] = float64(b.Count)
	}

	if len(commits) == 0 {
		return types.FrequencySignal{
			Type:           "commitFrequency",
			Score:          60,
			DailyBuckets:   dailyBuckets,
			Mean:           0,
			StdDev:         0,
			LongestGapDays: windowDays,
		}
	}

	mean := sumF(counts) / float64(len(counts))
	var variance float64
	for _, v := range counts {
		d := v - mean
		variance += d * d
	}
	sd := math.Sqrt(variance / float64(len(counts)))

	cv := 0.0
	if mean > 0 {
		cv = sd / mean
	}

	var cvNormalized float64
	switch {
	case cv < 0.5:
		cvNormalized = (cv / 0.5) * 20
	case cv < 1.0:
		cvNormalized = 20 + ((cv-0.5)/0.5)*20
	case cv < 1.5:
		cvNormalized = 40 + ((cv-1.0)/0.5)*20
	case cv < 2.5:
		cvNormalized = 60 + ((cv-1.5)/1.0)*20
	default:
		cvNormalized = 80 + math.Min(20, ((cv-2.5)/2.5)*20)
	}

	longestGap, currentGap := 0, 0
	for _, c := range counts {
		if c == 0 {
			currentGap++
			if currentGap > longestGap {
				longestGap = currentGap
			}
		} else {
			currentGap = 0
		}
	}

	gapScore := math.Min(60, float64(longestGap)/float64(windowDays)*100)
	score := int(math.Min(100, math.Round(cvNormalized*0.6+gapScore*0.4)))

	return types.FrequencySignal{
		Type:           "commitFrequency",
		Score:          score,
		DailyBuckets:   dailyBuckets,
		Mean:           round2(mean),
		StdDev:         round2(sd),
		LongestGapDays: longestGap,
	}
}

func sumF(xs []float64) float64 {
	var s float64
	for _, x := range xs {
		s += x
	}
	return s
}

func round2(x float64) float64 {
	return math.Round(x*100) / 100
}
