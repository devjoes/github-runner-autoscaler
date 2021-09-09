package scaling

import (
	"math"
	"math/rand"
	"strconv"
	"time"

	runnerv1alpha1 "github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	"k8s.io/klog/v2"
)

type Scaling struct {
	MinWorkers                int32   `json:"minWorkers"`
	MaxWorkers                int32   `json:"maxWorkers"`
	ScaleFactor               float64 `json:"scaleFactor"`
	Linear                    bool    `json:"linear"`
	ForceScaleUpWindowSecs    int32   `json:"forceScaleUpWindowSecs"`
	ForceScaleUpFrequencyDays int32   `json:"forceScaleUpFrequencyDays"`
}

func NewScaling(crd *runnerv1alpha1.ScaledActionRunner) Scaling {
	sf, _ := strconv.ParseFloat(*crd.Spec.ScaleFactor, 64)

	forceScaleUpWindowSecs := int32(20 * 60)
	forceScaleUpFrequencyDays := int32(20)
	if crd.Spec.Scaling != nil && crd.Spec.ForceScaleUpWindowSecs != nil {
		forceScaleUpWindowSecs = *crd.Spec.ForceScaleUpWindowSecs
	}
	if crd.Spec.Scaling != nil && crd.Spec.ForceScaleUpFrequencyDays != nil {
		forceScaleUpFrequencyDays = *crd.Spec.ForceScaleUpFrequencyDays
	}
	return Scaling{
		MinWorkers:                crd.Spec.MinRunners,
		MaxWorkers:                crd.Spec.MaxRunners,
		ForceScaleUpWindowSecs:    forceScaleUpWindowSecs,
		ForceScaleUpFrequencyDays: forceScaleUpFrequencyDays,
		ScaleFactor:               sf,
		Linear:                    sf == 0,
	}
}

func logistic(c float64, a float64, k float64, x float64) float64 {
	// https://www.desmos.com/calculator/agxuc5gip8
	const e = 2.718
	b := math.Pow(e, (0 - k))
	return c / (1 + a*math.Pow(b, x))
}

func (s *Scaling) GetOutput(queueLength int32) int32 {
	var result float64
	fMinWorkers := float64(s.MinWorkers)
	fMaxWorkers := float64(s.MaxWorkers)
	fQueueLength := float64(queueLength)

	result = fMinWorkers
	if queueLength > 0 {
		if s.Linear {
			result = fQueueLength
		} else {
			result = logistic(float64(s.MaxWorkers), float64(s.MaxWorkers), s.ScaleFactor, float64(queueLength))
		}

		if result > fMaxWorkers {
			result = fMaxWorkers
		}
		if result < fMinWorkers {
			result = fMinWorkers
		}
		if result == 0 && queueLength > 0 {
			result = 1
		}
	}
	klog.V(10).Infof("Scaling: queueLength=%d s.Linear=%t, s.MinWorkers=%d, s.MaxWorkers=%d, s.ScaleFactor=%f.  RESULT=%d (%f)", queueLength, s.Linear, s.MinWorkers, s.MaxWorkers, s.ScaleFactor, math.Round(result), result)
	return int32(math.Round(result))
}

func (s *Scaling) CalculateForcedScale(nextForcedScale *time.Time) (bool, time.Time) {
	if s.ForceScaleUpWindowSecs == 0 {
		return false, *nextForcedScale
	}
	rndSecs := rand.Intn(60 * 60 * 24)
	if nextForcedScale == nil {
		// If there is no next forced scale window then set it to some random time in the next 24 hours
		return false, time.Now().UTC().Add(time.Second * time.Duration(rndSecs))
	}

	endOfForcedScaleWindow := nextForcedScale.Add(time.Duration(s.ForceScaleUpWindowSecs) * time.Second)
	if endOfForcedScaleWindow.Before(time.Now().UTC()) {
		// If the forced scale window has ended then set a new one
		return false, time.Now().UTC().Add(time.Second * (time.Duration(s.ForceScaleUpFrequencyDays*24*60*60) + time.Duration(rndSecs)))
	} else if nextForcedScale.Before(time.Now().UTC()) {
		// If we are in the forced scale window then scale up
		return true, *nextForcedScale
	}

	return false, *nextForcedScale
}
