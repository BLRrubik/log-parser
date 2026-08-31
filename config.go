package main

import "flag"

type Config struct {
	InputDir   string
	OutputFile string
}

func ParseCommandLineArgs() (Config, error) {
	var (
		inputDir   string
		outputFile string
	)
	flag.StringVar(&inputDir, "input-dir", "logs", "Directory containing .log files")
	flag.StringVar(&outputFile, "output-file", "results.json", "JSON output file path")
	flag.Parse()

	return Config{
		InputDir:   inputDir,
		OutputFile: outputFile,
	}, nil
}
