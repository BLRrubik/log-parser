package parser

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

type LogEntry struct {
	Timestamp time.Time
	Level     string
	Service   string
	Message   string
	RequestID string
	UserID    string
}

func (e LogEntry) IsError() bool {
	return e.Level == "ERROR" || e.Level == "WARNING"
}

func (e LogEntry) String() string {
	return fmt.Sprintf(
		"%s [%s] %s: %s",
		e.Timestamp.Format("2006-01-02T15:04:05.000Z"),
		e.Level,
		e.Service,
		e.Message,
	)
}

func ParseLogLine(line string) (LogEntry, error) {
	pattern := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z)\s+\[(INFO|WARNING|ERROR)]\s+(\S+):\s([^,]+),\s+request_id=([^,]+),\s+user_id=(\S+)$`)
	matches := pattern.FindStringSubmatch(line)
	if len(matches) == 0 {
		return LogEntry{}, errors.New("could not parse Log Entry")
	}

	timestamp, err := time.Parse("2006-01-02T15:04:05.000Z", matches[1])
	if err != nil {
		return LogEntry{}, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	return LogEntry{
		Timestamp: timestamp,
		Level:     matches[2],
		Service:   matches[3],
		Message:   matches[4],
		RequestID: matches[5],
		UserID:    matches[6],
	}, nil
}
