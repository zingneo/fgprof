package fgprof

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/pprof/profile"
)

// Format decides how the output is rendered to the user.
type Format string

const (
	// FormatFolded is used by Brendan Gregg's FlameGraph utility, see
	// https://github.com/brendangregg/FlameGraph#2-fold-stacks.
	FormatFolded Format = "folded"
	// FormatPprof is used by Google's pprof utility, see
	// https://github.com/google/pprof/blob/master/proto/README.md.
	FormatPprof Format = "pprof"
)

func writeFormat(w io.Writer, s map[string]int, f Format, hz int, startTime, endTime time.Time) error {
	switch f {
	case FormatFolded:
		return writeFolded(w, s)
	case FormatPprof:
		return toPprof(s, hz, startTime, endTime).Write(w)
	default:
		return fmt.Errorf("unknown format: %q", f)
	}
}

func writeFolded(w io.Writer, s map[string]int) error {
	for _, stack := range sortedKeys(s) {
		count := s[stack]
		if _, err := fmt.Fprintf(w, "%s %d\n", stack, count); err != nil {
			return err
		}
	}
	return nil
}

func toPprof(s map[string]int, hz int, startTime, endTime time.Time) *profile.Profile {
	functionID := uint64(1)
	locationID := uint64(1)

	p := &profile.Profile{}
	m := &profile.Mapping{ID: 1, HasFunctions: true, HasLineNumbers: true, HasFilenames: true}
	p.Period = int64(1e9 / hz) // Number of nanoseconds between samples.
	p.TimeNanos = startTime.UnixNano()
	p.DurationNanos = int64(endTime.Sub(startTime))
	p.Mapping = []*profile.Mapping{m}
	p.SampleType = []*profile.ValueType{
		{
			Type: "samples",
			Unit: "count",
		},
		{
			Type: "time",
			Unit: "nanoseconds",
		},
	}
	p.PeriodType = &profile.ValueType{
		Type: "wallclock",
		Unit: "nanoseconds",
	}

	for _, stack := range sortedKeys(s) {
		count := s[stack]
		sample := &profile.Sample{
			Value: []int64{
				int64(count),
				int64(1000 * 1000 * 1000 / hz * count),
			},
		}
		for _, fnName := range strings.Split(stack, ";") {
			fl := strings.SplitN(fnName, ":", 3) // func, lineno, file
			line, _ := strconv.ParseInt(fl[1], 10, 64)
			// Serious lack of error handling...

			function := &profile.Function{
				ID:       functionID,
				Name:     fl[0],
				Filename: fl[2],
			}
			p.Function = append(p.Function, function)

			location := &profile.Location{
				ID:      locationID,
				Mapping: m,
				Line:    []profile.Line{{Function: function, Line: line}},
			}

			p.Location = append(p.Location, location)
			sample.Location = append([]*profile.Location{location}, sample.Location...)

			locationID++
			functionID++
		}
		p.Sample = append(p.Sample, sample)
	}
	return p
}

func sortedKeys(s map[string]int) []string {
	keys := make([]string, len(s))
	i := 0
	for stack := range s {
		keys[i] = stack
		i++
	}
	sort.Strings(keys)
	return keys
}
