package main

import (
	"fmt"
	"log"
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

func main() {
	logExpample := "2023-12-25T14:30:15.123Z [INFO] user-service: User authenticated, request_id=req_abc123, user_id=12345"

	entry, err := ParseLogLine(logExpample)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Parsed Log Entry:", entry)
}

func ParseLogLine(line string) (LogEntry, error) {
	pattern := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}.\d{3}Z)\s+\[(INFO|WARNING)]\s+(\S+):\s([^,]+),\s+request_id=([^,]+),\s+user_id=(\S+)$`)
	matches := pattern.FindStringSubmatch(line)

	timestamp, err := time.Parse(time.RFC3339, matches[1])
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
