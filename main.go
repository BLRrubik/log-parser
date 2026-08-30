package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
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
	_, err := ReadLogFile("sample.log")
	if err != nil {
		log.Fatal(err)
	}
}

func ReadLogFile(filepath string) ([]LogEntry, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var (
		totalLines int
		errorLines int
		entries    = make([]LogEntry, 0)
	)
	for scanner.Scan() {
		totalLines++

		entry, err := ParseLogLine(scanner.Text())
		if err != nil {
			errorLines++
			continue
		}

		entries = append(entries, entry)
	}

	fmt.Println("Total lines:", totalLines)
	fmt.Println("Error lines:", errorLines)
	fmt.Println("Successfully parsed lines:", totalLines-errorLines)

	return entries, scanner.Err()
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
