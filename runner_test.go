package performance_test

import (
	"context"
	"fmt"
	"math/rand"
	"performance"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/require"
)

func TestRunner(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	fn := func() error {
		// simulate latency: random sleep between 1 and 10 milliseconds
		time.Sleep(time.Duration(1+rand.Int31n(9)) * time.Millisecond)

		// simulate a 5% error rate.
		if (1 + rand.Int31n(99)) <= 5 {
			return fmt.Errorf("%s", gofakeit.Color())
		}

		return nil
	}

	seconds := 6
	callsPerSecond := 10

	runner := performance.NewRunner(time.Duration(seconds)*time.Second, fn, uint16(callsPerSecond), false)
	summary, err := runner.Run(ctx, nil)

	require.NoError(t, err)
	require.EqualValues(t, seconds*callsPerSecond, summary.Total)

	println(summary.String())
}
