package timeago

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type durationYMD struct {
	year  int
	month int
	day   int
}

var durationRegex = regexp.MustCompile(`(\d+)(\w)`)

// Parse takes a Duration string, parses and dedups it, returns a time object that is the
// current time subtracted by the duration provided. Supports typical time.ParseDuration format as well
// as day (d), month (m), and year (y)
func Parse(duration string) (time.Time, error) {
	t := time.Now()

	ymd, standardDuration, err := extractYmd(duration)
	if err != nil {
		return time.Time{}, errors.Wrapf(err, "Extracting year, month, and day from duration: %s", duration)
	}

	if standardDuration != "" {
		d, err := time.ParseDuration(fmt.Sprintf("-%s", standardDuration))
		if err != nil {
			return time.Time{}, errors.Wrapf(err, "Parsing provided duration string: %s", duration)
		}
		t = t.Add(d)
	}

	return t.AddDate(-ymd.year, -ymd.month, -ymd.day), nil
}

func extractYmd(duration string) (*durationYMD, string, error) {
	ymd := durationYMD{}
	durationMatches := durationRegex.FindAllStringSubmatch(duration, -1)
	standardDurations := []string{}

	for _, d := range durationMatches {
		full, key := d[0], d[2]
		amount, err := strconv.Atoi(d[1])
		if err != nil {
			return nil, "", errors.Wrapf(err, "parsing duration for %s", full)
		}

		switch key {
		case "y":
			ymd.year += amount
		case "M":
			ymd.month += amount
		case "d":
			ymd.day += amount
		default:
			standardDurations = append(standardDurations, full)
		}
	}

	return &ymd, strings.Join(standardDurations, ""), nil
}
