// Package rules implements the game of Komikan: legal moves, captures, turns,
// and victory.
//
// The package knows nothing about HTTP or the browser. It works on a board
// from internal/board and a position, and every rule that varies between
// variants of the game lives in the Ruleset.
package rules

import (
	"errors"
	"fmt"
)

// LayoutStandard is the layout id of the canonical board.
const LayoutStandard = "komikan-standard"

// Ruleset holds the values that vary between variants of the game.
//
// The defaults describe the variant documented in docs/overview.md. Every
// value is configurable so a variant is a ruleset, not a fork of the engine.
type Ruleset struct {
	// PanguiCapturesToWin is how many Trewa the Pangui must capture to win.
	PanguiCapturesToWin int `json:"panguiCapturesToWin"`
	// AllowMultipleCaptures lets the Pangui chain captures in one turn.
	AllowMultipleCaptures bool `json:"allowMultipleCaptures"`
	// MultipleCapturesMandatory forces the Pangui to keep capturing while a
	// capture is available.
	MultipleCapturesMandatory bool `json:"multipleCapturesMandatory"`
	// TrewaCanMoveBackward lets the Trewa move away from the triangle as
	// well as towards it.
	TrewaCanMoveBackward bool `json:"trewaCanMoveBackward"`
	// PanguiMovesFirst starts the game with the Pangui.
	PanguiMovesFirst bool `json:"panguiMovesFirst"`
	// PuTrewaWinByBlocking makes the Trewa win when the Pangui has no
	// legal move.
	PuTrewaWinByBlocking bool `json:"puTrewaWinByBlocking"`
	// PuTrewaWinByOccupyingTriangle is reserved for variants with a
	// designated triangle area. The canonical board has none, so the
	// engine does not implement it.
	PuTrewaWinByOccupyingTriangle bool `json:"puTrewaWinByOccupyingTriangle"`
	// CapturePreferred makes the Pangui capture whenever it can. With it
	// off, the Pangui may decline a capture and move normally instead.
	CapturePreferred bool `json:"capturePreferred"`
	// DrawWhenTrewaCannotMove ends the game in a draw when the Trewa have
	// no legal move.
	DrawWhenTrewaCannotMove bool `json:"drawWhenTrewaCannotMove"`
	// ThresholdBeforeBlocking checks the capture threshold before the
	// blocking condition, so the capture that wins the game is not
	// cancelled by the position it leaves behind.
	ThresholdBeforeBlocking bool `json:"thresholdBeforeBlocking"`
}

// DefaultRuleset returns the ruleset of the documented default variant.
//
// The values are answers to the open rule issues where an answer has been
// given, and proposals where it has not. They live in one place, so a
// different answer changes this function and nothing else.
func DefaultRuleset() Ruleset {
	return Ruleset{
		PanguiCapturesToWin:           6,
		AllowMultipleCaptures:         true,
		MultipleCapturesMandatory:     true,
		TrewaCanMoveBackward:          true,
		PanguiMovesFirst:              true,
		PuTrewaWinByBlocking:          true,
		PuTrewaWinByOccupyingTriangle: false,
		CapturePreferred:              false,
		DrawWhenTrewaCannotMove:       true,
		ThresholdBeforeBlocking:       true,
	}
}

// Validate checks that the ruleset makes sense for a board.
func (r Ruleset) Validate() error {
	if r.PanguiCapturesToWin < 1 {
		return fmt.Errorf("rules: the capture threshold must be at least 1, got %d", r.PanguiCapturesToWin)
	}
	if r.MultipleCapturesMandatory && !r.AllowMultipleCaptures {
		return errors.New("rules: mandatory continuation needs multiple captures to be allowed")
	}
	return nil
}
