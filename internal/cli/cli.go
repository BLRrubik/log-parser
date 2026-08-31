package cli

import (
	"errors"
	"flag"
	"os"
)

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

	if _, err := os.Stat(inputDir); err != nil && os.IsNotExist(err) {
		return Config{}, errors.New("input directory does not exist")
	}

	return Config{
		InputDir:   inputDir,
		OutputFile: outputFile,
	}, nil
}
