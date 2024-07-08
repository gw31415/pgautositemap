package utils

import "slices"

func findLIS(positions []int) []int {
	n := len(positions)
	dp := make([]int, n)
	prev := make([]int, n)
	for i := range dp {
		dp[i] = 1
		prev[i] = -1
	}

	for i := 0; i < n; i++ {
		for j := 0; j < i; j++ {
			if positions[j] < positions[i] && dp[j]+1 > dp[i] {
				dp[i] = dp[j] + 1
				prev[i] = j
			}
		}
	}

	lisLength := 0
	lisIndex := 0
	for i := 0; i < n; i++ {
		if dp[i] > lisLength {
			lisLength = dp[i]
			lisIndex = i
		}
	}

	lis := []int{}
	for lisIndex != -1 {
		lis = append(lis, lisIndex)
		lisIndex = prev[lisIndex]
	}

	// 逆順になっているので反転する
	slices.Reverse(lis)

	return lis
}

// positions に対して、位置が昇順になるように調整するための増減値を返す
func AdjustPositions(positions []int) []int {
	lisIndices := findLIS(positions)
	lisSet := make(map[int]bool)
	for _, index := range lisIndices {
		lisSet[index] = true
	}

	n := len(positions)
	adjustments := make([]int, n)
	expectedValue := 1

	for i := 0; i < n; i++ {
		if lisSet[i] {
			expectedValue = positions[i]
		} else {
			for slices.Contains(positions, expectedValue) {
				expectedValue++
			}
			adjustments[i] = expectedValue - positions[i]
			expectedValue++
		}
	}

	return adjustments
}
