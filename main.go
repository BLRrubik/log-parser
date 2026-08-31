package main

import (
	"context"
	"fmt"
	"log"
	"log-parser/internal/cli"
	"log-parser/internal/processor"
	"log-parser/internal/reporter"
	"log-parser/internal/scanner"
	"os"
	"os/signal"
	"time"
)

func main() {
	cfg, err := cli.ParseCommandLineArgs()
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now()

	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	go func() {
		<-sigChan
		fmt.Println("Получен сигнал прерывания...")
		cancel()
	}()

	filePaths, err := scanner.ScanLogDirectory(cfg.InputDir)
	if err != nil {
		log.Fatal(err)
	}

	entries, err := processor.ProcessFilesConcurrently(ctx, filePaths, 2)
	if err != nil {
		log.Fatal(err)
	}

	requests := processor.CorrelateRequests(entries)
	failedRequestIDs := processor.DetectFailedRequests(requests)

	analysisResult := reporter.AnalysisResult{
		TotalEntriesProcessed: len(entries),
		FailedRequestsFound:   len(failedRequestIDs),
		FailedRequests:        make([]reporter.FailedRequestReport, 0, len(failedRequestIDs)),
	}
	for _, request := range failedRequestIDs {
		failedEntries := processor.SortTimelineByTimestamp(requests[request])
		firstEntry, ok := processor.FindFirstFailure(failedEntries)
		if !ok {
			continue
		}

		timeLine := make([]string, 0, len(failedEntries))
		for _, entry := range failedEntries {
			timeLine = append(timeLine, entry.String())
		}

		failedReport := reporter.FailedRequestReport{
			RequestID:      request,
			FailingService: firstEntry.Service,
			ErrorMessage:   firstEntry.Message,
			Timeline:       timeLine,
		}

		analysisResult.FailedRequests = append(analysisResult.FailedRequests, failedReport)
	}

	processingTime := time.Since(now).Seconds()
	analysisResult.ProcessingTimeSeconds = processingTime

	if err = reporter.WriteJSONReport(analysisResult, cfg.OutputFile); err != nil {
		log.Fatal(err)
	}
}
