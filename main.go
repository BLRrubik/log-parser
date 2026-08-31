package main

import (
	"log"
	"time"
)

func main() {
	cfg, err := ParseCommandLineArgs()
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now()
	filePaths, err := ScanLogDirectory(cfg.InputDir)
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
		TotalEntriesProcessed: len(entries),
		FailedRequestsFound:   len(failedRequestIDs),
		FailedRequests:        make([]FailedRequestReport, 0, len(failedRequestIDs)),
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

	processingTime := time.Since(now).Seconds()
	analysisResult.ProcessingTimeSeconds = processingTime

	if err = WriteJSONReport(analysisResult, cfg.OutputFile); err != nil {
		log.Fatal(err)
	}
}
