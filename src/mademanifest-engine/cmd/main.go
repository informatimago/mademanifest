package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"mademanifest-engine/pkg/engine"
)

func main() {

	const version = "phase-4 http-service version 0.1"

	canonDirFlag := flag.String("canon-directory", "canon", "canon directory path")
	flag.StringVar(canonDirFlag, "cd", "canon", "canon directory path")

	gateSequenceFlag := flag.String("gate-sequence-file", "", "gate sequence file path")
	flag.StringVar(gateSequenceFlag, "gs", "", "gate sequence file path")

	mandalaConstantsFlag := flag.String("mandala-constants-file", "", "mandala constants file path")
	flag.StringVar(mandalaConstantsFlag, "mc", "", "mandala constants file path")

	nodePolicyFlag := flag.String("node-policy-file", "", "node policy file path")
	flag.StringVar(nodePolicyFlag, "np", "", "node policy file path")

	helpFlag := flag.Bool("help", false, "print usage")
	flag.BoolVar(helpFlag, "h", false, "print usage")

	versionFlag := flag.Bool("version", false, "print version")
	flag.BoolVar(versionFlag, "v", false, "print version")

	dosFlag := flag.Bool("dos", false, "write CRLF line endings to output file")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s \\\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "           [--canon-directory|-cd        $canon_directory] \\\n")
		fmt.Fprintf(os.Stderr, "           [--gate-sequence-file|-gs     $gate_sequence_file] \\\n")
		fmt.Fprintf(os.Stderr, "           [--mandala-constants-file|-mc $mandala_constants_file] \\\n")
		fmt.Fprintf(os.Stderr, "           [--node-policy-file|-np       $node_policy_file] \\\n")
		fmt.Fprintf(os.Stderr, "           [--dos] [--help|-h] [--version|-v] \\\n")
		fmt.Fprintf(os.Stderr, "           $inputFile $outputFile\n")
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	if *versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}

	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(1)
	}

	inputFile := flag.Arg(0)
	outputFile := flag.Arg(1)

	canonPaths, err := engine.ResolveCanonPaths(*canonDirFlag, *gateSequenceFlag, *mandalaConstantsFlag, *nodePolicyFlag)
	if err != nil {
		log.Fatalf("Failed to resolve canon paths: %v", err)
	}

	file, err := os.Open(inputFile)
	if err != nil {
		log.Fatalf("Failed to open JSON file: %v", err)
	}
	defer file.Close()

	output, err := engine.Run(file, canonPaths)
	if err != nil {
		log.Fatalf("Failed to process input JSON: %v", err)
	}
	outputJSON, err := engine.Render(output, *dosFlag)
	if err != nil {
		log.Fatalf("Failed to marshal output to JSON: %v", err)
	}

	if err := os.WriteFile(outputFile, outputJSON, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write file: %v\n", err)
		os.Exit(1)
	}
}
