//go:build !race

package repository

// raceDetectorEnabled reports whether the test binary was built with -race.
// Go exposes no runtime flag for this, so it is resolved with a build-tag pair
// (see race_enabled_test.go). Test-only: the constant never enters a
// production build.
const raceDetectorEnabled = false
