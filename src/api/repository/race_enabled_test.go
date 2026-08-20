//go:build race

package repository

// raceDetectorEnabled reports whether the test binary was built with -race.
// See race_disabled_test.go for the other half of the pair.
const raceDetectorEnabled = true
