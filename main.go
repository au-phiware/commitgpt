package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

//go:embed prompt.txt
var promptData string

var (
	Endpoint         = "https://api.anthropic.com/v1/messages"
	AnthropicVersion = "2023-06-01"
	Model            = "claude-3-haiku-20240307"
	MaxTokens        = 2048

	MillionInputTokensUnitPrice  = 0.25
	MillionOutputTokensUnitPrice = 1.25
)

func TransformText(r io.Reader) (tx io.Reader) {
	var w *io.PipeWriter
	tx, w = io.Pipe()

	go func() {
		defer w.Close()
		buf := make([]byte, 1024)
		lookback := make([]byte, 2)

		n, err := r.Read(lookback)
		if err != nil {
			if err != io.EOF {
				w.CloseWithError(err)
			}
			return
		}
		if n == 1 {
			n, err = r.Read(lookback[1:])
			if err != nil {
				if err != io.EOF {
					w.CloseWithError(err)
				}
				return
			}
		}

		_, err = w.Write(lookback)
		if err != nil {
			w.CloseWithError(err)
			return
		}

		for {
			n, err := r.Read(buf)
			if err != nil {
				if err != io.EOF {
					w.CloseWithError(err)
				}
				break
			}

			var i int
			for j := 0; j < n; j++ {
				b := buf[j]
				if string(lookback) == ":\n" && strings.ContainsRune("*+-", rune(b)) {
					buf[j] = '\n'
					_, err = w.Write(buf[i : j+1])
					if err != nil {
						w.CloseWithError(err)
						return
					}
					buf[j] = b
					i = j
				}
				lookback[0] = lookback[1]
				lookback[1] = b
			}
			w.Write(buf[i:n])
		}
	}()

	return
}

func formatWarning(warning, content string) string {
	cmd := exec.Command("pandoc", "--columns=70", "-t", "gfm")
	cmd.Stdin = TransformText(strings.NewReader(content))
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "pandoc:", err)
		return ""
	}
	formattedWarning := strings.ReplaceAll(out.String(), "\n", "\n# ")
	return fmt.Sprintf("# **%s**\n# \n# %s\n", warning, formattedWarning)
}

func formatPlain(content string) string {
	cmd := exec.Command("pandoc", "--columns=72", "-t", "gfm")
	cmd.Stdin = TransformText(strings.NewReader(content))
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error running pandoc:", err)
		return ""
	}
	return fmt.Sprintf("%s\n", out.String())
}

func makeAPICall(diff string) (_ string, err error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return
	}

	content := fmt.Sprintf(promptData, diff)

	data := map[string]interface{}{
		"model":      Model,
		"max_tokens": MaxTokens,
		"messages": []map[string]string{
			{"role": "user", "content": content},
		},
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", Endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", AnthropicVersion)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResponse struct {
			Type  string `json:"type"`
			Error struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResponse)
		if errResponse.Type == "" {
			err = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			return
		}
		err = fmt.Errorf("%s: %s: %s", errResponse.Type, errResponse.Error.Type, errResponse.Error.Message)
		return
	}

	var apiResponse struct {
		Id      string `json:"id"`
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	err = json.NewDecoder(resp.Body).Decode(&apiResponse)
	if err != nil {
		return
	}

	if apiResponse.StopReason != "end_turn" {
		err = fmt.Errorf("unexpected stop reason: %s", apiResponse.StopReason)
		return
	}

	if len(apiResponse.Content) == 0 || apiResponse.Content[0].Text == "" {
		err = fmt.Errorf("no response from model")
		return
	}

	sensitiveWarn, largeFilesWarn, thought, commitMessage := extractMessages(apiResponse.Content[0].Text)

	var response strings.Builder
	if sensitiveWarn != "" {
		response.WriteString(formatWarning("Sensitive Information Warning", sensitiveWarn))
	}
	if largeFilesWarn != "" {
		response.WriteString(formatWarning("Large Files Warning", largeFilesWarn))
	}
	if commitMessage != "" {
		response.WriteString(formatPlain(commitMessage))
	}

	response.WriteString("# ------------------------ >8 ------------------------\n")
	response.WriteString("# Do not modify or remove the line above.\n")
	response.WriteString("# Everything below it will be ignored.\n")
	response.WriteString("#\n")
	response.WriteString(fmt.Sprintf("# API ID: %s\n", apiResponse.Id))
	response.WriteString(fmt.Sprintf("# Input tokens: %d ($%.4f)\n", apiResponse.Usage.InputTokens, float64(apiResponse.Usage.InputTokens)*MillionInputTokensUnitPrice/1e6))
	response.WriteString(fmt.Sprintf("# Output tokens: %d ($%.4f)\n", apiResponse.Usage.OutputTokens, float64(apiResponse.Usage.OutputTokens)*MillionOutputTokensUnitPrice/1e6))
	response.WriteString("#\n")

	if thought != "" {
		response.WriteString("# Below is the thought process that created the above message.\n")
		response.WriteString(formatPlain(thought))
		response.WriteString("\n")
	}

	return strings.TrimSuffix(response.String(), "\n"), nil
}

func extractMessages(apiResponse string) (string, string, string, string) {
	parts := map[string]strings.Builder{}
	var builder *strings.Builder
	lines := strings.Split(apiResponse, "\n")
	for _, line := range lines {
		if line == "<sensitive-info-warning>" {
			builder = &strings.Builder{}
		} else if line == "</sensitive-info-warning>" {
			parts["sensitive-info-warning"] = *builder
			builder = nil
		} else if line == "<large-files-warning>" {
			builder = &strings.Builder{}
		} else if line == "</large-files-warning>" {
			parts["large-files-warning"] = *builder
			builder = nil
		} else if line == "<thinkthrough>" {
			builder = &strings.Builder{}
		} else if line == "</thinkthrough>" {
			parts["thought"] = *builder
			builder = nil
		} else if line == "<commit-message>" {
			builder = &strings.Builder{}
		} else if line == "</commit-message>" {
			parts["commit-message"] = *builder
			builder = nil
		} else if builder != nil {
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	}
	sensitiveWarn := parts["sensitive-info-warning"]
	largeFilesWarn := parts["large-files-warning"]
	thought := parts["thought"]
	commitMessage := parts["commit-message"]
	return strings.TrimSuffix(sensitiveWarn.String(), "\n"),
		strings.TrimSuffix(largeFilesWarn.String(), "\n"),
		strings.TrimSuffix(thought.String(), "\n"),
		strings.TrimSuffix(commitMessage.String(), "\n")
}

func handleVerboseContent(content string) string {
	lines := strings.Split(content, "\n")

	var verboseContent strings.Builder
	var i int
	for i = 0; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "# Please enter the commit message for your changes.") {
			break
		}
	}
	for i = i + 1; i < len(lines); i++ {
		if lines[i] == "#" {
			break
		}
	}
	for i = i + 1; i < len(lines); i++ {
		if lines[i] == "# ------------------------ >8 ------------------------" {
			i += 2
			continue
		}
		verboseContent.WriteString(lines[i])
		verboseContent.WriteString("\n")
	}

	return strings.TrimSuffix(verboseContent.String(), "\n")
}

func main() {
	commitMsgFile := os.Args[1]
	commitSource := os.Args[2]

	skip := os.Getenv("SKIP_PREPARE_COMMIT_MSG")
	if v, err := strconv.ParseBool(skip); skip != "" && (err != nil || v) {
		os.Exit(0)
	}

	if _, err := os.Stat(commitMsgFile); os.IsNotExist(err) {
		os.Exit(0)
	}

	if commitSource != "" && commitSource != "template" {
		os.Exit(0)
	}

	content, err := os.ReadFile(commitMsgFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	trailer := handleVerboseContent(string(content))

	diff, err := exec.Command("git", "diff", "--cached").Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	apiResponse, err := makeAPICall(string(diff))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if apiResponse != "" {
		err := os.WriteFile(commitMsgFile, []byte(apiResponse+"\n"+trailer), 0644)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
