package scaling

import (
	"math"
	"strconv"

	runnerv1alpha1 "github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	"k8s.io/klog/v2"
)

type Scaling struct {
	MinWorkers  int32   `json:"minWorkers"`
	MaxWorkers  int32   `json:"maxWorkers"`
	ScaleFactor float64 `json:"scaleFactor"`
	Linear      bool    `json:"linear"`
}

func NewScaling(crd *runnerv1alpha1.ScaledActionRunner) Scaling {
	sf, _ := strconv.ParseFloat(*crd.Spec.ScaleFactor, 64)

	return Scaling{
		MinWorkers:  crd.Spec.MinRunners,
		MaxWorkers:  crd.Spec.MaxRunners,
		ScaleFactor: sf,
		Linear:      sf == 0,
	}
}

func logistic(c float64, a float64, k float64, x float64) float64 {
	// https://www.desmos.com/calculator/agxuc5gip8
	const e = 2.718
	b := math.Pow(k, (0 - e))
	return c / (1 + a*math.Pow(b, x))
}

func (s *Scaling) GetOutput(queueLength int32) int32 {
	var result int32

	result = s.MinWorkers
	if queueLength > 0 {
		if s.Linear {
			result = queueLength
		} else {
			result = int32(math.Round(logistic(float64(s.MaxWorkers), float64(s.MaxWorkers), s.ScaleFactor, float64(queueLength))))
		}
		if result > s.MaxWorkers {
			result = s.MaxWorkers
		}
		if result < s.MinWorkers {
			result = s.MinWorkers
		}
		if result == 0 && queueLength > 0 {
			result = 1
		}
	}
	klog.V(10).Infof("Scaling: queueLength=%d s.Linear=%t, s.MinWorkers=%f, s.MaxWorkers=%f, s.ScaleFactor=%f.  RESULT=%f", queueLength, s.Linear, s.MinWorkers, s.MaxWorkers, s.ScaleFactor, result)
	return result
}
