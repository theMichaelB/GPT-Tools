package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
	"github.com/tiktoken-go/tokenizer"
)

func parseLLMInclude(filePath string) (*ignore.GitIgnore, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var patterns []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return ignore.CompileIgnoreLines(patterns...), nil
}

func checkFiles(gitIgnore *ignore.GitIgnore, rootDir string) ([]string, error) {
	var matched []string
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !info.IsDir() && gitIgnore.MatchesPath(path) {
			matched = append(matched, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return matched, nil
}

func processPromptFiles(folderPath string, w *bufio.Writer) error {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return err
	}
	fileNumber := 1
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".txt") && e.Name() != "instructions.txt" {
			content, err := os.ReadFile(filepath.Join(folderPath, e.Name()))
			if err != nil {
				log.Printf("Error reading file %s: %v", e.Name(), err)
				continue
			}
			fmt.Fprintf(w, "<meta prompt %d>\n%s\n</meta prompt %d>\n", fileNumber, string(content), fileNumber)
			fileNumber++
		}
	}
	return nil
}

func includeInstructionsFile(folderPath string, w *bufio.Writer) error {
	instructionsPath := filepath.Join(folderPath, "instructions.txt")
	if _, err := os.Stat(instructionsPath); err == nil {
		content, err := os.ReadFile(instructionsPath)
		if err != nil {
			return fmt.Errorf("error reading instructions.txt: %w", err)
		}
		fmt.Fprintf(w, "<user_instructions>\n%s\n</user_instructions>\n", string(content))
	}
	return nil
}

func printFileStats(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(f)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	fmt.Printf("File Statistics for %s:\nSize: %d bytes\nNumber of lines: %d\n", filePath, info.Size(), lineCount)

	enc, err := tokenizer.Get(tokenizer.O200kBase)
	if err != nil {
		panic(err)
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	ids, _, _ := enc.Encode(string(content))
	fmt.Printf("Token count: %d\n", len(ids))

	return nil
}

func main() {
	gitignorePath := ".llminclude"
	rootDir := "."

	gitIgnore, err := parseLLMInclude(gitignorePath)
	if err != nil {
		log.Fatalf("Error parsing .llminclude: %v", err)
	}

	matchedFiles, err := checkFiles(gitIgnore, rootDir)
	if err != nil {
		log.Fatalf("Error checking files: %v", err)
	}

	outputFile, err := os.Create("prompt.txt")
	if err != nil {
		log.Fatalf("Error creating prompt.txt: %v", err)
	}
	defer outputFile.Close()
	writer := bufio.NewWriter(outputFile)
	defer writer.Flush()

	if len(matchedFiles) > 0 {
		fmt.Println("Matched files:")
		for _, file := range matchedFiles {
			fmt.Println(file)
			content, err := os.ReadFile(file)
			if err != nil {
				log.Printf("Error reading file %s: %v", file, err)
				continue
			}
			relativePath, err := filepath.Rel(rootDir, file)
			if err != nil {
				log.Printf("Error getting relative path for %s: %v", file, err)
				continue
			}
			ext := strings.TrimPrefix(filepath.Ext(file), ".")
			fmt.Fprintf(writer, "<file_contents>\nFile: %s\n```%s\n%s\n```\n</file_contents>\n", relativePath, ext, string(content))
		}
	} else {
		fmt.Println("No files matched.")
	}

	promptsFolder := "prompts"
	if err := includeInstructionsFile(promptsFolder, writer); err != nil {
		log.Fatalf("Error including instructions.txt: %v", err)
	}

	if err := processPromptFiles(promptsFolder, writer); err != nil {
		log.Fatalf("Error processing prompts folder: %v", err)
	}

	if err := printFileStats("prompt.txt"); err != nil {
		log.Printf("Error printing file statistics: %v", err)
	}
}
