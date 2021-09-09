package scaling

import (
	"testing"
	"time"

	runnerv1alpha1 "github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func getScaling() Scaling {
	one := "1"
	crd := runnerv1alpha1.ScaledActionRunner{
		Spec: runnerv1alpha1.ScaledActionRunnerSpec{
			MaxRunners:  10,
			MinRunners:  1,
			ScaleFactor: &one,
		},
	}

	return NewScaling(&crd)
}

func TestMinWorkersWhenNoWork(t *testing.T) {
	s := getScaling()
	assert.Equal(t, s.MinWorkers, s.GetOutput(0))
}
func TestLinearWhenSet(t *testing.T) {
	s := getScaling()
	s.Linear = true
	for i := int32(1); i <= s.MaxWorkers; i++ {
		if assert.Equal(t, i, s.GetOutput(i)) {
			break
		}
	}
	assert.Equal(t, s.MaxWorkers, s.GetOutput(s.MaxWorkers*2))
}

func testRangeAgainstOutputs(t *testing.T, maxWorkers int, scaleFactor float64, expected []int) {
	s := getScaling()
	s.MaxWorkers = int32(maxWorkers)
	s.ScaleFactor = scaleFactor
	for i, r := range expected {
		if assert.Equal(t, int32(r), s.GetOutput(int32(i)), "%d should result in %d", i, r) {
			break
		}
	}
}

func TestLogisticBounds(t *testing.T) {
	s := Scaling{
		MinWorkers:  0,
		MaxWorkers:  4,
		ScaleFactor: 0.5,
		Linear:      false,
	}
	assert.Equal(t, int32(4), s.GetOutput(30))
	assert.Equal(t, int32(1), s.GetOutput(1))
	s.MinWorkers = 2
	assert.Equal(t, int32(2), s.GetOutput(1))
}

func TestLogistic(t *testing.T) {
	testRangeAgainstOutputs(t, 10, 0.25, []int{1, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 7, 7, 8, 8, 8, 9, 9, 9, 9, 10})
	testRangeAgainstOutputs(t, 10, 0.5, []int{1, 1, 2, 3, 4, 5, 7, 8, 8, 9, 9, 10})
	testRangeAgainstOutputs(t, 10, 1, []int{1, 2, 4, 7, 8, 9, 10})
	testRangeAgainstOutputs(t, 100, 0.5, []int{1, 2, 3, 4, 7, 11, 17, 25, 35, 47, 60, 71, 80, 87, 92, 95, 97, 98, 99, 99, 100})
	testRangeAgainstOutputs(t, 100, 1, []int{1, 3, 7, 17, 35, 60, 80, 92, 97, 99, 100})
}

func TestCalculateForcedScale_ShouldReturnNextForceScale_IfNil(t *testing.T) {
	s := Scaling{
		ForceScaleUpWindow:    time.Duration(20) * time.Minute,
		ForceScaleUpFrequency: time.Duration(20*24) * time.Hour,
	}
	scaleNow, nextForcedScale := s.CalculateForcedScale(nil)
	assert.False(t, scaleNow)
	assert.NotNil(t, nextForcedScale)
	assert.True(t, nextForcedScale.After(time.Now().UTC()))
	assert.True(t, nextForcedScale.Before(time.Now().UTC().Add(time.Hour*time.Duration(24))))
}

func TestCalculateForcedScale_ShouldReturnNextForceScale_IfBeforeScaleWindow(t *testing.T) {
	s := Scaling{
		ForceScaleUpWindow:    time.Duration(20) * time.Minute,
		ForceScaleUpFrequency: time.Duration(20*24) * time.Hour,
	}
	yesterday := time.Now().UTC().Add(time.Hour * time.Duration(-24))
	scaleNow, nextForcedScale := s.CalculateForcedScale(&yesterday)
	assert.False(t, scaleNow)
	assert.NotNil(t, nextForcedScale)
	assert.True(t, nextForcedScale.After(time.Now().UTC().Add(s.ForceScaleUpFrequency)))
}

func TestCalculateForcedScale_ShouldReturnTrueScaleNow_IfInScalingWindow(t *testing.T) {
	s := Scaling{
		ForceScaleUpWindow:    time.Duration(20) * time.Minute,
		ForceScaleUpFrequency: time.Duration(20*24) * time.Hour,
	}
	fiveMinsAgo := time.Now().UTC().Add(time.Minute * time.Duration(-5))
	scaleNow, nextForcedScale := s.CalculateForcedScale(&fiveMinsAgo)
	assert.True(t, scaleNow)
	assert.Equal(t, fiveMinsAgo, nextForcedScale)
}

func TestCalculateForcedScale_ShouldDoNothing_IfScalingWindowIsInFuture(t *testing.T) {
	s := Scaling{
		ForceScaleUpWindow:    time.Duration(20) * time.Minute,
		ForceScaleUpFrequency: time.Duration(20*24) * time.Hour,
	}
	hourInFuture := time.Now().UTC().Add(time.Hour)
	scaleNow, nextForcedScale := s.CalculateForcedScale(&hourInFuture)
	assert.False(t, scaleNow)
	assert.Equal(t, hourInFuture, nextForcedScale)
}

func TestCalculateForcedScale_ShouldDoNothing_IfScalingIsDisabled(t *testing.T) {
	s := Scaling{
		ForceScaleUpWindow:    time.Duration(0),
		ForceScaleUpFrequency: time.Duration(20*24) * time.Hour,
	}
	now := time.Now().UTC()
	scaleNow, nextForcedScale := s.CalculateForcedScale(&now)
	assert.False(t, scaleNow)
	assert.Equal(t, now, nextForcedScale)
}
