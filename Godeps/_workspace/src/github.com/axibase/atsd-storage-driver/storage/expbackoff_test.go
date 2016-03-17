package storage

import (
	"math/rand"
	"testing"
	"time"
)

func TestExpandBackoff(t *testing.T) {
	src := rand.NewSource(time.Now().UTC().UnixNano())
	randGen := rand.New(src)

	limit := time.Duration(randGen.Int63n(450)) * time.Hour
	expBackoff := NewExpBackoff(1*time.Microsecond, limit)

	for counter := 0; ; {
		d := expBackoff.Duration()

		if d > limit {
			t.Error("duration is greater than limit: duration=", d, "limit=", limit)
		}
		if d == limit {
			if counter > 10 {
				break
			}
			counter++
		}
	}
}
