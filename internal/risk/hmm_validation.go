package risk

import (
	"log"
	"math"
)

type HMMValidationResult struct {
	Checks []HMMCheck
	Passed bool
}

type HMMCheck struct {
	Name    string
	Passed  bool
	Message string
}

func ValidateHMMParams(transition [4][4]float64, means [4]float64, sds [4]float64) HMMValidationResult {
	var checks []HMMCheck

	highVolIdx := 2
	crisisIdx := 3
	trendingIdx := 1

	if sds[highVolIdx] < sds[0] || sds[highVolIdx] < sds[trendingIdx] {
		checks = append(checks, HMMCheck{
			Name:    "emission_sd_ordering",
			Passed:  false,
			Message: "HIGH_VOL emission SD should not be the smallest — states may be mislabeled",
		})
	} else {
		checks = append(checks, HMMCheck{Name: "emission_sd_ordering", Passed: true, Message: "SD ordering looks sensible"})
	}

	if transition[highVolIdx][crisisIdx] < 0.001 {
		checks = append(checks, HMMCheck{
			Name:    "transition_highvol_to_crisis",
			Passed:  false,
			Message: "HIVOL→CRISIS transition is near-zero — model cannot detect crisis escalation from high volatility",
		})
	} else {
		checks = append(checks, HMMCheck{Name: "transition_highvol_to_crisis", Passed: true, Message: "HIVOL→CRISIS transition is plausible"})
	}

	if transition[trendingIdx][trendingIdx] < 0.5 {
		checks = append(checks, HMMCheck{
			Name:    "transition_trending_self",
			Passed:  false,
			Message: "TRENDING self-transition is low — trending state is transient, not persistent",
		})
	} else {
		checks = append(checks, HMMCheck{Name: "transition_trending_self", Passed: true, Message: "TRENDING self-transition is plausible"})
	}

	for i := range transition {
		rowSum := 0.0
		for j := range transition[i] {
			rowSum += transition[i][j]
		}
		if math.Abs(rowSum-1.0) > 0.01 {
			checks = append(checks, HMMCheck{
				Name:    "transition_row_sum",
				Passed:  false,
				Message: "Transition matrix row does not sum to 1.0",
			})
			break
		}
	}

	for _, sd := range sds {
		if sd <= 0 {
			checks = append(checks, HMMCheck{
				Name:    "emission_sd_positive",
				Passed:  false,
				Message: "Emission SD must be positive",
			})
			break
		}
	}

	passed := true
	for _, c := range checks {
		if !c.Passed {
			passed = false
		}
	}
	result := HMMValidationResult{Checks: checks, Passed: passed}

	for _, c := range checks {
		if !c.Passed {
			log.Printf("HMM validation WARNING: %s — %s", c.Name, c.Message)
		}
	}

	return result
}
