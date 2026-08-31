package processor

import (
	"bufio"
	"fmt"
	"log-parser/internal/parser"
	"os"
	"sort"
	"sync"
)

func ProcessFilesConcurrently(filePaths []string, numWorkers int) ([]parser.LogEntry, error) {
	jobs := make(chan string, 100)
	results := make(chan []parser.LogEntry, 100)

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fileWorker(jobs, results)
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for _, filePath := range filePaths {
		jobs <- filePath
	}

	close(jobs)

	var entries []parser.LogEntry
	for result := range results {
		entries = append(entries, result...)
	}

	return entries, nil
}

func fileWorker(jobs <-chan string, results chan<- []parser.LogEntry) {
	for path := range jobs {
		entries, _ := ReadLogFile(path)
		results <- entries
	}
}

func ProcessMultipleFiles(filePaths []string) ([]parser.LogEntry, error) {
	var entries []parser.LogEntry

	for _, filePath := range filePaths {
		fileEntries, err := ReadLogFile(filePath)
		if err != nil {
			continue
		}

		entries = append(entries, fileEntries...)
	}

	return entries, nil
}

func CorrelateRequests(entries []parser.LogEntry) map[string][]parser.LogEntry {
	mp := make(map[string][]parser.LogEntry)
	for _, entry := range entries {
		mp[entry.RequestID] = append(mp[entry.RequestID], entry)
	}

	return mp
}

func DetectFailedRequests(correlatedRequests map[string][]parser.LogEntry) []string {
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

func FindFirstFailure(requestEntries []parser.LogEntry) (parser.LogEntry, bool) {
	sort.Slice(requestEntries, func(i, j int) bool {
		return requestEntries[i].Timestamp.Before(requestEntries[j].Timestamp)
	})

	for _, entry := range requestEntries {
		if entry.IsError() {
			return entry, true
		}
	}

	return parser.LogEntry{}, false
}

func SortTimelineByTimestamp(entries []parser.LogEntry) []parser.LogEntry {
	copied := make([]parser.LogEntry, len(entries))
	copy(copied, entries)

	sort.Slice(copied, func(i, j int) bool {
		return copied[i].Timestamp.Before(copied[j].Timestamp)
	})

	return copied
}

func ReadLogFile(filepath string) ([]parser.LogEntry, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var (
		totalLines int
		errorLines int
		entries    = make([]parser.LogEntry, 0)
	)
	for scanner.Scan() {
		totalLines++

		entry, err := parser.ParseLogLine(scanner.Text())
		if err != nil {
			errorLines++
			continue
		}

		entries = append(entries, entry)
	}

	return entries, scanner.Err()
}
