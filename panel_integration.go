package main

import (
	"fmt"
	"math"
	"github.com/holoviz/panel"
	"github.com/Celebrum/base-calculation-for-anti-entropy"
)

// CreatePanel creates and displays the panel for calculating anti-entropy growth rate.
func CreatePanel() {
	// Create a new panel
	p := panel.New()

	// Add widgets for input values
	df := panel.NewFloatSlider("Fractal Dimension (Df)", 0, 10, 1)
	v := panel.NewFloatSlider("Electric Potential (V)", 0, 100, 1)
	c := panel.NewFloatSlider("Composition of Solvent (C)", 0, 1, 0.1)
	mu := panel.NewFloatSlider("Mobility of Ions/Particles (μ)", 0, 1, 0.1)

	// Add a button to calculate the growth rate
	calculateButton := panel.NewButton("Calculate Growth Rate")
	calculateButton.OnClick(func() {
		growthRate := CalculateGrowthRate(df.Value(), v.Value(), c.Value(), mu.Value())
		fmt.Printf("Growth Rate: %f\n", growthRate)
	})

	// Add widgets to the panel
	p.Add(df, v, c, mu, calculateButton)

	// Display the panel
	p.Show()
}

// CalculateGrowthRate calculates the fractal growth rate.
func CalculateGrowthRate(df, v, c, mu float64) float64 {
	k := 1.0 // constant
	return k * df * mu * v
}

// CalculateFractalFibonacci calculates the fractal Fibonacci sequence.
func CalculateFractalFibonacci(n int) []int {
	if n <= 0 {
		return []int{}
	}
	sequence := make([]int, n)
	sequence[0] = 0
	if n > 1 {
		sequence[1] = 1
	}
	for i := 2; i < n; i++ {
		sequence[i] = sequence[i-1] + sequence[i-2]
	}
	return sequence
}

// CalculateAntiEntropyRate calculates the anti-entropy rate.
func CalculateAntiEntropyRate(growthRate, entropyCoefficient, entropyOverTime float64) float64 {
	return growthRate - entropyCoefficient*entropyOverTime
}

// CalculateZ calculates the value of Z based on the given parameters.
func CalculateZ(w, x, y, z, c float64) float64 {
	return ((w + x - y*z) / y) + math.Pow(z, 3) + math.Pow(c, 3)*math.Pow(1.61803398875, 2)
}
