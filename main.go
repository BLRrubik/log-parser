package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
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

type AnalysisResult struct {
	TotalEntries          int                   `json:"total_entries_processed"`
	FailedCount           int                   `json:"failed_requests_found"`
	ProcessingTimeSeconds float64               `json:"processing_time_seconds"`
	FailedRequests        []FailedRequestReport `json:"failed_requests"`
}

type FailedRequestReport struct {
	RequestID      string   `json:"request_id"`
	FailingService string   `json:"failing_service"`
	ErrorMessage   string   `json:"error_message"`
	Timeline       []string `json:"timeline"`
}

func main() {
	now := time.Now()
	filePaths, err := ScanLogDirectory("logs")
	if err != nil {
		log.Fatal(err)
	}

	entries, err := ProcessMultipleFiles(filePaths)
	if err != nil {
		log.Fatal(err)
	}

	requests := CorrelateRequests(entries)
	failedRequestIDs := DetectFailedRequests(requests)

	analysisResult := AnalysisResult{
		TotalEntries:   len(entries),
		FailedCount:    len(failedRequestIDs),
		FailedRequests: make([]FailedRequestReport, 0, len(failedRequestIDs)),
	}
	for _, request := range failedRequestIDs {
		failedEntries := SortTimelineByTimestamp(requests[request])
		firstEntry, ok := FindFirstFailure(failedEntries)
		if !ok {
			continue
		}

		timeLine := make([]string, 0, len(failedEntries))
		for _, entry := range failedEntries {
			timeLine = append(timeLine, entry.String())
		}

		failedReport := FailedRequestReport{
			RequestID:      request,
			FailingService: firstEntry.Service,
			ErrorMessage:   firstEntry.Message,
			Timeline:       timeLine,
		}

		analysisResult.FailedRequests = append(analysisResult.FailedRequests, failedReport)
	}

	if entry, ok := FindFirstFailure(requests[failedRequestIDs[0]]); ok {
		fmt.Println("first failed request ID", entry)
	}

	processingTime := time.Since(now).Seconds()
	analysisResult.ProcessingTimeSeconds = processingTime

	if err = WriteJSONReport(analysisResult, "result.json"); err != nil {
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

func ScanLogDirectory(dirPath string) ([]string, error) {
	var logFiles []string

	filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".log") {
			logFiles = append(logFiles, path)
		}
		return nil
	})

	return logFiles, nil
}

func ProcessMultipleFiles(filePaths []string) ([]LogEntry, error) {
	var entries []LogEntry

	for _, filePath := range filePaths {
		fileEntries, err := ReadLogFile(filePath)
		if err != nil {
			continue
		}

		entries = append(entries, fileEntries...)
	}

	return entries, nil
}

func CorrelateRequests(entries []LogEntry) map[string][]LogEntry {
	mp := make(map[string][]LogEntry)
	for _, entry := range entries {
		mp[entry.RequestID] = append(mp[entry.RequestID], entry)
	}

	return mp
}

func DetectFailedRequests(correlatedRequests map[string][]LogEntry) []string {
	var failedRequests []string

	for requestID, entries := range correlatedRequests {
		for _, entry := range entries {
			if entry.IsError() {
				failedRequests = append(failedRequests, requestID)

				break
			}
		}
	}

	return failedRequests
}

func FindFirstFailure(requestEntries []LogEntry) (LogEntry, bool) {
	sort.Slice(requestEntries, func(i, j int) bool {
		return requestEntries[i].Timestamp.Before(requestEntries[j].Timestamp)
	})

	for _, entry := range requestEntries {
		if entry.IsError() {
			return entry, true
		}
	}

	return LogEntry{}, false
}

func SortTimelineByTimestamp(entries []LogEntry) []LogEntry {
	copied := make([]LogEntry, len(entries))
	copy(copied, entries)

	sort.Slice(copied, func(i, j int) bool {
		return copied[i].Timestamp.Before(copied[j].Timestamp)
	})

	return copied
}

func WriteJSONReport(result AnalysisResult, filename string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal analysis result: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}
