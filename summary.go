package performance

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Delta456/box-cli-maker/v2"
)

// Summary is a performance result report.
type Summary struct {
	// Total number of iterations.
	Total uint64

	// Time is the amount of time spent running the test.
	Time time.Duration

	// Failed is the number of failed iterations.
	Failed uint64

	// FailedPercent percentage of failed iterations.
	FailedPercent float64

	// Success is the number of successful iterations.
	Success uint64

	// SuccessPercent percentage of successful iterations.
	SuccessPercent float64

	// Errors is a list of error messages and their respective counts detected
	// during the test.
	Errors map[string]uint64

	// Latencies the list of observed latencies (in Milliseconds) indexed by
	// percentile.
	Latencies map[uint8]int64
}

func (s Summary) String() string {
	lines := make([]string, 0)

	lines = append(lines,
		fmt.Sprintf("- Running Time: %s", s.Time),
		fmt.Sprintf("- Iterations: %d", s.Total),
		fmt.Sprintf("- Success ✔: %d (%.2f%%)", s.Success, s.SuccessPercent),
		fmt.Sprintf("- Failures ✘: %d (%.2f%%)", s.Failed, s.FailedPercent),
		"- Latencies:",
	)

	latencies := make([]string, 0, len(s.Latencies))

	for p, obs := range s.Latencies {
		latencies = append(latencies, fmt.Sprintf("  - p(%d) = %d ms", p, obs))
	}

	sort.Strings(latencies)

	lines = append(lines, latencies...)

	if s.FailedPercent > 0 {
		lines = append(lines, "- Errors:")
		errs := make([]string, 0)

		for err, count := range s.Errors {
			errs = append(errs, fmt.Sprintf("  - [#%d] %s", count, err))
		}

		sort.Strings(errs)

		lines = append(lines, errs...)
	}

	infoBox := box.New(box.Config{Px: 2, Py: 2, Type: "Double", TitlePos: "Top", Color: "Green"})

	return infoBox.String("RESULTS", strings.Join(lines, "\n"))
}
