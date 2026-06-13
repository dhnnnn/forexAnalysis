package indicators

import (
	"math"
	"testing"
)

func TestCalcBollingerBands_Basic(t *testing.T) {
	// 20 known close prices
	closes := []float64{
		1.1000, 1.1050, 1.1020, 1.1080, 1.1060,
		1.1100, 1.1090, 1.1120, 1.1070, 1.1040,
		1.1030, 1.1060, 1.1080, 1.1110, 1.1090,
		1.1050, 1.1070, 1.1100, 1.1130, 1.1150,
	}

	result := CalcBollingerBands(closes, 20, 2.0)

	// Middle should be SMA of all 20 closes
	expectedMiddle := 0.0
	for _, c := range closes {
		expectedMiddle += c
	}
	expectedMiddle /= 20.0

	if math.Abs(result.Middle-expectedMiddle) > 1e-10 {
		t.Errorf("Middle: got %f, want %f", result.Middle, expectedMiddle)
	}

	// Upper > Middle > Lower
	if result.Upper <= result.Middle {
		t.Errorf("Upper (%f) should be > Middle (%f)", result.Upper, result.Middle)
	}
	if result.Lower >= result.Middle {
		t.Errorf("Lower (%f) should be < Middle (%f)", result.Lower, result.Middle)
	}

	// BBPosition should be in [0, 1]
	if result.BBPosition < 0.0 || result.BBPosition > 1.0 {
		t.Errorf("BBPosition out of range: %f", result.BBPosition)
	}
}

func TestCalcBollingerBands_ZeroBandwidth(t *testing.T) {
	// All identical prices → zero stddev → zero bandwidth
	closes := make([]float64, 20)
	for i := range closes {
		closes[i] = 1.5000
	}

	result := CalcBollingerBands(closes, 20, 2.0)

	if result.Upper != result.Lower {
		t.Errorf("Expected upper == lower for constant prices, got Upper=%f, Lower=%f", result.Upper, result.Lower)
	}
	if result.BBPosition != 0.50 {
		t.Errorf("Expected BBPosition = 0.50 for zero bandwidth, got %f", result.BBPosition)
	}
}

func TestCalcBollingerBands_InsufficientData(t *testing.T) {
	closes := []float64{1.0, 1.1, 1.2} // only 3, need 20

	result := CalcBollingerBands(closes, 20, 2.0)

	if result.Upper != 0 || result.Middle != 0 || result.Lower != 0 || result.BBPosition != 0 {
		t.Errorf("Expected zero result for insufficient data, got %+v", result)
	}
}

func TestCalcBollingerBands_CloseAboveUpper(t *testing.T) {
	// Construct data where the last close is far above the upper band
	// 19 values around 1.0, last value is 2.0
	closes := make([]float64, 20)
	for i := 0; i < 19; i++ {
		closes[i] = 1.0
	}
	closes[19] = 2.0

	result := CalcBollingerBands(closes, 20, 2.0)

	// BBPosition should be clamped to 1.0
	if result.BBPosition != 1.0 {
		t.Errorf("Expected BBPosition = 1.0 when close is above upper band, got %f", result.BBPosition)
	}
}

func TestCalcBollingerBands_CloseBelowLower(t *testing.T) {
	// Construct data where the last close is far below the lower band
	// 19 values around 2.0, last value is 0.5
	closes := make([]float64, 20)
	for i := 0; i < 19; i++ {
		closes[i] = 2.0
	}
	closes[19] = 0.5

	result := CalcBollingerBands(closes, 20, 2.0)

	// BBPosition should be clamped to 0.0
	if result.BBPosition != 0.0 {
		t.Errorf("Expected BBPosition = 0.0 when close is below lower band, got %f", result.BBPosition)
	}
}

func TestCalcBollingerBands_BandSymmetry(t *testing.T) {
	closes := []float64{
		1.1000, 1.1050, 1.1020, 1.1080, 1.1060,
		1.1100, 1.1090, 1.1120, 1.1070, 1.1040,
		1.1030, 1.1060, 1.1080, 1.1110, 1.1090,
		1.1050, 1.1070, 1.1100, 1.1130, 1.1150,
	}

	result := CalcBollingerBands(closes, 20, 2.0)

	// Bands should be symmetric around middle
	upperDist := result.Upper - result.Middle
	lowerDist := result.Middle - result.Lower

	if math.Abs(upperDist-lowerDist) > 1e-10 {
		t.Errorf("Bands not symmetric: upper dist=%f, lower dist=%f", upperDist, lowerDist)
	}
}
