package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// type application struct {
// 	openAIClient *openai.Client
// }

type result struct {
	chapterId int
	data      string
	error     error
}

func main() {

	apiKey := flag.String("t", "", "token")
	filePath := flag.String("fP", "", "Path to text file")
	maxConcurrent := flag.Int("mC", 5, "Max concurrency at a time. Default is 5")

	flag.Parse()
	fileContent, err := readFileContent(*filePath)

	if err != nil {
		fmt.Print(err)
		return
	}

	fileContent = removeBlankLines(fileContent)

	chapters := splitIntoChapters(fileContent)

	jobs := make(chan int, len(chapters))
	apiTasks := make(chan result, len(chapters))

	client := openai.NewClient(*apiKey)

	for w := 1; w <= *maxConcurrent; w++ {
		go callSummary(client, jobs, chapters, apiTasks)
	}

	for idx := 0; idx < len(chapters); idx++ {
		jobs <- idx
	}
	close(jobs)

	results := make([]result, len(chapters))

	for i := range results {
		results[i] = <-apiTasks
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].chapterId < results[j].chapterId
	})

	for _, summaryInfo := range results {
		fmt.Printf("%+v\n\n", summaryInfo.data)
	}

}

func readFileContent(filePath string) (string, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return string(file[:]), nil
}

func splitIntoChapters(text string) []string {
	headingPattern := regexp.MustCompile(`(?m)^[A-Z0-9?,!'\s]+$(?:\n|\z)`)
	headings := headingPattern.FindAllString(text, -1)

	chapterContents := headingPattern.Split(text, -1)

	var chapters []string
	for i, heading := range headings {
		chapterContent := ""
		if i+1 < len(chapterContents) {
			chapterContent = strings.TrimSpace(chapterContents[i+1])
		}
		chapters = append(chapters, fmt.Sprintf("%s\n%s", strings.TrimSpace(heading), chapterContent))
	}

	return chapters

}

func removeBlankLines(text string) string {
	blankLinePattern := regexp.MustCompile(`(?m)^\s*\n`)
	return blankLinePattern.ReplaceAllString(text, "")
}

func callSummary(client *openai.Client, jobs <-chan int, chapters []string, resultChan chan<- result) {
	for chapId := range jobs {
		resp, err := client.CreateChatCompletion(context.Background(),
			openai.ChatCompletionRequest{
				Model: openai.GPT3Dot5Turbo,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleSystem,
						Content: "Give me summary of this chapter",
					},
					{
						Role:    openai.ChatMessageRoleUser,
						Content: chapters[chapId],
					},
				},
			})
		if err != nil {
			fmt.Printf("ChatCompletion error: %v\n", err)
			resultChan <- result{
				chapterId: chapId,
				data:      "Cant summary this chapter",
				error:     err,
			}
			return
		}

		resultChan <- result{
			chapterId: chapId,
			data:      resp.Choices[0].Message.Content,
			error:     nil,
		}
	}

}
