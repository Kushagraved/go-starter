package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// rupeesToPaisa parses a decimal rupee string (up to two decimal places) into an
// integer paisa amount. "100" -> 10000, "33.34" -> 3334, "0.5" -> 50.
//
// It rejects negatives, more than two decimal places, and non-numeric input —
// the parser must not silently truncate money.
func rupeesToPaisa(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty amount")
	}
	if strings.HasPrefix(s, "-") {
		return 0, errors.Errorf("amount cannot be negative: %q", s)
	}

	whole, frac, hasDot := strings.Cut(s, ".")

	rupees, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, errors.Errorf("invalid amount: %q", s)
	}

	var paisa int64
	if hasDot {
		if len(frac) == 0 || len(frac) > 2 {
			return 0, errors.Errorf("amount must have at most two decimal places: %q", s)
		}
		// Pad "5" -> "50" so both halves are exactly two digits of paisa.
		if len(frac) == 1 {
			frac += "0"
		}
		p, err := strconv.ParseInt(frac, 10, 64)
		if err != nil {
			return 0, errors.Errorf("invalid amount: %q", s)
		}
		paisa = p
	}

	return rupees*100 + paisa, nil
}

// percentToBasisPoints parses a percent string (up to two decimals) into basis
// points: "33.33" -> 3333, "50" -> 5000, "100" -> 10000. This keeps the
// sum-to-100 check exact integer arithmetic.
func percentToBasisPoints(s string) (int64, error) {
	// A percent with two decimals scales to basis points exactly like rupees
	// scale to paisa, so reuse the same conversion.
	bp, err := rupeesToPaisa(s)
	if err != nil {
		return 0, errors.Wrap(err, "invalid percent")
	}
	return bp, nil
}

// paisaToRupees formats an integer paisa amount as a two-decimal rupee string:
// 3334 -> "33.34", 10000 -> "100.00", 50 -> "0.50".
func paisaToRupees(paisa int64) string {
	sign := ""
	if paisa < 0 {
		sign = "-"
		paisa = -paisa
	}
	return fmt.Sprintf("%s%d.%02d", sign, paisa/100, paisa%100)
}
