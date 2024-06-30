package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"

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
	var wg sync.WaitGroup

	filePath := os.Args[1]

	fileContent, err := readFileContent(filePath)

	if err != nil {
		fmt.Print(err)
		return
	}

	fileContent = removeBlankLines(fileContent)

	chapters := splitIntoChapters(fileContent)

	ch := make(chan result)

	client := openai.NewClient("")

	for idx, chapter := range chapters {
		wg.Add(1)
		go func(idx int, chapter string) {
			defer wg.Done()
			callSummary(client, idx+1, chapter, ch)
		}(idx, chapter)
	}

	results := make([]result, len(chapters))

	for i := range results {
		results[i] = <-ch
	}
	wg.Wait()

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

func callSummary(client *openai.Client, chapId int, chapterContent string, ch chan<- result) {
	resp, err := client.CreateChatCompletion(context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "Hello. Can you give me summary of each chapter",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: chapterContent,
				},
			},
		})
	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		ch <- result{
			chapterId: chapId,
			data:      "Cant summary this chapter",
			error:     err,
		}
		return
	}

	ch <- result{
		chapterId: chapId,
		data:      resp.Choices[0].Message.Content,
		error:     nil,
	}

}
