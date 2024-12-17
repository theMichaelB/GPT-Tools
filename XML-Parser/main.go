package main

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"strings"
)

type Files struct {
	XMLName xml.Name `xml:"files"`
	Files   []File   `xml:"file"`
}

type File struct {
	ChangeSummary string `xml:"change_summary"`
	Content       string `xml:"content,omitempty"`
	Operation     string `xml:"operation,attr"`
	Language      string `xml:"language,attr"`
	Path          string `xml:"path,attr"`
}

func main() {
	// Open the file
	file, err := os.Open("output.txt")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Scanner to read the file line by line
	scanner := bufio.NewScanner(file)
	foundMeta, foundXML := false, false
	var jsonLines, xmlLines []string
	openBraces := 0
	isCollectingJSON := false

	for scanner.Scan() {
		line := scanner.Text()

		// Look for ===META===
		if strings.TrimSpace(line) == "===META===" {
			foundMeta = true
			continue
		}

		// Look for ===XML===
		if strings.TrimSpace(line) == "===XML===" {
			foundXML = true
			continue
		}

		// Collect JSON
		if foundMeta && !foundXML {
			trimmed := strings.TrimSpace(line)
			if strings.Contains(trimmed, "{") {
				isCollectingJSON = true
			}
			if isCollectingJSON {
				jsonLines = append(jsonLines, line)
				openBraces += strings.Count(trimmed, "{")
				openBraces -= strings.Count(trimmed, "}")
				if openBraces == 0 {
					isCollectingJSON = false
					foundMeta = false
				}
			}
		}

		// Collect XML
		if foundXML {
			xmlLines = append(xmlLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	// Handle JSON
	if len(jsonLines) > 0 {
		fmt.Println("Parsing JSON:")
		jsonString := strings.Join(jsonLines, "\n")
		var data map[string]interface{}
		err = json.Unmarshal([]byte(jsonString), &data)
		if err != nil {
			fmt.Println("Error parsing JSON:", err)
			return
		}
		printJSON(data, 0)
	}

	// Handle XML
	if len(xmlLines) > 0 {
		fmt.Println("\nParsing XML:")
		xmlString := strings.Join(xmlLines, "\n")
		var files Files
		err = xml.Unmarshal([]byte(xmlString), &files)
		if err != nil {
			fmt.Println("Error parsing XML:", err)
			return
		}

		// Process each file based on the operation
		for _, file := range files.Files {
			switch strings.ToUpper(file.Operation) {
			case "CREATE":
				createFile(file)
			case "UPDATE":
				updateFile(file)
			case "DELETE":
				deleteFile(file)
			default:
				fmt.Printf("Unknown operation: %s\n", file.Operation)
			}
		}
	}
}

// Helper function to print JSON recursively
func printJSON(data interface{}, indent int) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			fmt.Printf("%s%s:\n", strings.Repeat("  ", indent), key)
			printJSON(value, indent+1)
		}
	case []interface{}:
		for i, value := range v {
			fmt.Printf("%s[%d]:\n", strings.Repeat("  ", indent), i)
			printJSON(value, indent+1)
		}
	default:
		fmt.Printf("%s%v\n", strings.Repeat("  ", indent), v)
	}
}

func createFile(file File) {
	fmt.Printf("Creating file: %s\n", file.Path)

	// Ensure the directory structure exists
	dir := getDirFromPath(file.Path)
	if dir != "" {
		err := os.MkdirAll(dir, 0755) // Create directories with proper permissions
		if err != nil {
			fmt.Printf("Error creating directories: %s\n", err)
			return
		}
	}

	// Check if the file already exists
	if _, err := os.Stat(file.Path); err == nil {
		fmt.Printf("File already exists: %s. Skipping creation.\n", file.Path)
		return
	}

	// Create the file
	err := os.WriteFile(file.Path, []byte(file.Content), 0644)
	if err != nil {
		fmt.Printf("Error creating file: %s\n", err)
	}
}

// Helper function to extract the directory path from a file path
func getDirFromPath(filePath string) string {
	lastSlash := strings.LastIndex(filePath, "/")
	if lastSlash == -1 {
		return "" // No directory structure in the path
	}
	return filePath[:lastSlash]
}

func updateFile(file File) {
	fmt.Printf("Updating file: %s\n", file.Path)
	if _, err := os.Stat(file.Path); os.IsNotExist(err) {
		fmt.Printf("File not found: %s\n", file.Path)
		return
	}
	err := os.WriteFile(file.Path, []byte(file.Content), 0644)
	if err != nil {
		fmt.Printf("Error updating file: %s\n", err)
	}
}

func deleteFile(file File) {
	fmt.Printf("Deleting file: %s\n", file.Path)
	err := os.Remove(file.Path)
	if err != nil {
		fmt.Printf("Error deleting file: %s\n", err)
	}
}
