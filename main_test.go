package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_extractMessages(t *testing.T) {
	tests := []struct {
		name  string
		args  string
		want  string
		want1 string
		want2 string
		want3 string
	}{
		{
			name: "test 1",
			args: `
<sensitive-info-warning>
Warning: The diff contains sensitive information:
- Line 5: Hardcoded password found in the code

Please remove the sensitive information from the code and commit again.
</sensitive-info-warning>
			`,
			want: `Warning: The diff contains sensitive information:
- Line 5: Hardcoded password found in the code

Please remove the sensitive information from the code and commit again.`,
		},
		{
			name: "test 2",
			args: `
<large-files-warning>
Warning: The diff contains files larger than 50MB:
- data/large_file.bin (120MB)

Consider using Git Large File Storage (LFS) for these files. Learn more at https://git-lfs.github.com
</large-files-warning>
`,
			want1: `Warning: The diff contains files larger than 50MB:
- data/large_file.bin (120MB)

Consider using Git Large File Storage (LFS) for these files. Learn more at https://git-lfs.github.com`,
		},
		{
			name: "test 3",
			args: `<thinkthrough>
Please think through the changes you are making. Make sure they are necessary and that you have considered the impact of the changes on the project.
</thinkthrough>`,
			want2: `Please think through the changes you are making. Make sure they are necessary and that you have considered the impact of the changes on the project.`,
		},
		{
			name: "test 4",
			args: `
<commit-message>
feat: support custom AssertExpectedCalls implementation

Prior to this change, a mock can be anything AssertExpectedCalls cannot.

This change allows a mock to hook in and override the
AssertExpectedCalls behaviour for whatever reason.
</commit-message>
`,
			want3: `feat: support custom AssertExpectedCalls implementation

Prior to this change, a mock can be anything AssertExpectedCalls cannot.

This change allows a mock to hook in and override the
AssertExpectedCalls behaviour for whatever reason.`,
		},
		{
			name: "test 5",
			args: `
<large-files-warning>
Warning: The diff contains files larger than 50MB:
- data/large_file.bin (120MB)

Consider using Git Large File Storage (LFS) for these files. Learn more at https://git-lfs.github.com
</large-files-warning>
<commit-message>
feat: support custom AssertExpectedCalls implementation

Prior to this change, a mock can be anything AssertExpectedCalls cannot.

This change allows a mock to hook in and override the
AssertExpectedCalls behaviour for whatever reason.
</commit-message>
`,
			want1: `Warning: The diff contains files larger than 50MB:
- data/large_file.bin (120MB)

Consider using Git Large File Storage (LFS) for these files. Learn more at https://git-lfs.github.com`,
			want3: `feat: support custom AssertExpectedCalls implementation

Prior to this change, a mock can be anything AssertExpectedCalls cannot.

This change allows a mock to hook in and override the
AssertExpectedCalls behaviour for whatever reason.`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, got2, got3 := extractMessages(tt.args)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("extractMessages() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want1, got1); diff != "" {
				t.Errorf("extractMessages() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want2, got2); diff != "" {
				t.Errorf("extractMessages() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want3, got3); diff != "" {
				t.Errorf("extractMessages() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_TransformText(t *testing.T) {
	tests := []struct {
		name string
		args string
		want string
	}{
		{
			name: "test 1",
			args: `Warning: The diff contains sensitive information:
- Line 5: Hardcoded password found in the code

Please remove the sensitive information from the code and commit again.`,
			want: `Warning: The diff contains sensitive information:

- Line 5: Hardcoded password found in the code

Please remove the sensitive information from the code and commit again.`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got strings.Builder
			_, err := io.Copy(&got, TransformText(strings.NewReader(tt.args)))
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(tt.want, got.String()); diff != "" {
				t.Errorf("TransformText() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_formatWarning(t *testing.T) {
	type args struct {
		title string
		text  string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test 1",
			args: args{
				title: "Sensitive Information Warning",
				text: `Warning: The diff contains sensitive information:

- Line 5: Hardcoded password found in the code

Please remove the sensitive information from the code and commit again.
`,
			},
			want: `# **Sensitive Information Warning**
# 
# Warning: The diff contains sensitive information:
# 
# - Line 5: Hardcoded password found in the code
# 
# Please remove the sensitive information from the code and commit
# again.
# 
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatWarning(tt.args.title, tt.args.text)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("formatWarning() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_formatPlain(t *testing.T) {
	type args struct {
		text string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test 1",
			args: args{
				text: `feat: support custom AssertExpectedCalls implementation

Prior to this change, a mock can be anything AssertExpectedCalls cannot.

This change allows a mock to hook in and override the AssertExpectedCalls behaviour for whatever reason.
`,
			},
			want: `feat: support custom AssertExpectedCalls implementation

Prior to this change, a mock can be anything AssertExpectedCalls cannot.

This change allows a mock to hook in and override the
AssertExpectedCalls behaviour for whatever reason.

`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatPlain(tt.args.text)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("formatPlain() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_handleVerboseContent(t *testing.T) {
	tests := []struct {
		name string
		args string
		want string
	}{
		{
			name: "test 1",
			args: `# If applied, this commit will...

# Why is this change needed?
Prior to this change, 

# How does it address the issue?
This change

# Provide links to any relevant tickets, articles or other resources


# Please enter the commit message for your changes. Lines starting
# with '#' will be ignored, and an empty message aborts the commit.
#
# On branch main
# Your branch is up to date with 'origin/main'.
#
# Changes to be committed:
#	modified:   backend/package.json
#
# Untracked files:
#	.husky/prepare-commit-msg
#	backend/package-lock.json
#
# ------------------------ >8 ------------------------
# Do not modify or remove the line above.
# Everything below it will be ignored.
diff --git a/backend/package.json b/backend/package.json
index 4f6b7e7..b1b3b3e 100644
`,
			want: `# On branch main
# Your branch is up to date with 'origin/main'.
#
# Changes to be committed:
#	modified:   backend/package.json
#
# Untracked files:
#	.husky/prepare-commit-msg
#	backend/package-lock.json
#
diff --git a/backend/package.json b/backend/package.json
index 4f6b7e7..b1b3b3e 100644
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handleVerboseContent(tt.args)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("handleVerboseContent() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_makeAPICall(t *testing.T) {
	tests := []struct {
		name string
		args string
		want string
		err  string
	}{
		{
			name: "test 1",
			args: `diff --git a/backend/package.json b/backend/package.json
index 4f6b7e7..b1b3b3e 100644`,
			want: `# **Sensitive Information Warning**
# 
# The provided diff does not contain any sensitive information like API
# keys, passwords, or personal data.
# 
# **Large Files Warning**
# 
# The provided diff does not contain any files larger than 50MB.
# 
feat: Introduce dynamic worker interval configuration

This change introduces the ability to dynamically configure the worker
intervals for various workflows using AWS Systems Manager (SSM)
parameters.

Prior to this change, the worker intervals were hardcoded in the
application. This made it difficult to adjust the intervals for
different environments (e.g., dev, test, staging, production) without
modifying the code and redeploying the application.

The key changes include:

- Introduced new SSM parameters to control the batch size and interval
  for the following workflows:
  - ACT
  - Initial
  - Plate Type
  - IPS Send
  - IPS Prepare
- Updated the worker implementations to use the
  worker.NewSSMOverrideInterval function, which allows the worker
  interval to be dynamically overridden by the corresponding SSM
  parameter.
- Ensured consistent naming of the SSM parameter keys across the
  different workflows.
- Fixed an issue with the plate type workflow parameter name.

These changes provide more flexibility in configuring the worker
intervals and batch sizes for different environments, without the need
to modify the application code. This will help improve the application’s
performance and resource utilization in various deployment scenarios.

# ------------------------ >8 ------------------------
# Do not modify or remove the line above.
# Everything below it will be ignored.
#
# API ID: 12345
# Input tokens: 10 ($0.0000)
# Output tokens: 5 ($0.0000)
#
# Below is the thought process that created the above message.
The changes in the provided diff appear to be related to the
configuration and implementation of various worker components in a
microservice application. The overall purpose of the changes seems to
be:

1.  Introducing more flexibility and configurability in the worker
    intervals and batch sizes for different workflows, such as the IPS
    poller, ACT workflow, initial workflow, plate type workflow, and IPS
    send workflow.
2.  Refactoring the worker implementations to use SSM (AWS Systems
    Manager) parameters to dynamically override the default interval
    values, instead of using static intervals.
3.  Ensuring consistent naming and organization of the SSM parameter
    keys across the different workflows.
4.  Addressing a potential issue with the plate type workflow parameter
    name.

The changes do not appear to contain any sensitive information like API
keys, passwords, or personal data. The diff also does not include any
large files over 50MB that would require the use of Git LFS.

The specific modifications made in each file are as follows:

1.  cfn/ssm.yaml:
    - Introduced new SSM parameters to control the batch size and
      interval for various workflows, such as ACT, initial, plate type,
      IPS send, and IPS prepare.
    - Updated the existing IPS poller parameters to use the new mapping
      structure.
2.  internal/worker/workflowAct/worker.go:
    - Introduced a new constant IntervalKey to hold the SSM parameter
      key for the ACT worker interval.
    - Updated the worker creation to use the
      worker.NewSSMOverrideInterval function, which allows the interval
      to be dynamically overridden by the SSM parameter.
3.  internal/worker/workflowInitial/worker.go:
    - Introduced a new constant IntervalKey to hold the SSM parameter
      key for the initial worker interval.
    - Updated the worker creation to use the
      worker.NewSSMOverrideInterval function.
4.  internal/worker/workflowIps/worker.go:
    - Introduced a new constant IntervalKey to hold the SSM parameter
      key for the IPS worker interval.
    - Updated the worker creation to use the
      worker.NewSSMOverrideInterval function.
5.  internal/worker/workflowIpsSend/worker.go:
    - Introduced a new constant IntervalKey to hold the SSM parameter
      key for the IPS send worker interval.
    - Updated the worker creation to use the
      worker.NewSSMOverrideInterval function.
6.  internal/worker/workflowPlateType/config.go:
    - Updated the plateTypeBatchSizeKey constant to use the correct SSM
      parameter key.
7.  internal/worker/workflowPlateType/worker.go:
    - Introduced a new constant IntervalKey to hold the SSM parameter
      key for the plate type worker interval.
    - Updated the worker creation to use the
      worker.NewSSMOverrideInterval function.

The changes seem to be aimed at improving the configurability and
flexibility of the worker components, allowing the batch sizes and
intervals to be adjusted based on the environment (e.g., dev, test,
staging, production) without the need to modify the code directly. This
approach can be beneficial for managing the application’s performance
and resource utilization in different environments.

The refactoring to use SSM parameters for interval configuration is a
reasonable choice, as it allows the values to be easily updated without
redeploying the application. The consistent naming of the SSM parameter
keys across the different workflows also helps maintain code readability
and maintainability.

Overall, the changes appear to be well-considered and aimed at improving
the application’s flexibility and configurability.

`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer newHTTPTestServer(t, tt.args)()
			got, err := makeAPICall(tt.args)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("makeAPICall() mismatch (-want +got):\n%s", diff)
			}
			if err != nil && err.Error() != tt.err {
				t.Errorf("makeAPICall() err = %v, want %v", err, tt.err)
			}
		})
	}
}

func newHTTPTestServer(t *testing.T, diff string) func() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("unexpected URL: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("x-api-key") != "test-api-key" {
			t.Errorf("unexpected x-api-key: %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("unexpected anthropic-version: %s", r.Header.Get("anthropic-version"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected Content-Type: %s", r.Header.Get("Content-Type"))
		}
		var data map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			t.Error(err)
		}
		if data["model"] != "claude-3-haiku-20240307" {
			t.Errorf("unexpected model: %s", data["model"])
		}
		if fmt.Sprint(data["max_tokens"]) != "2048" {
			t.Errorf("unexpected max_tokens: %v", data["max_tokens"])
		}
		messages, ok := data["messages"].([]interface{})
		if !ok {
			t.Errorf("unexpected messages type: %T", data["messages"])
		}
		if len(messages) != 1 {
			t.Errorf("unexpected messages length: %d", len(messages))
		}
		message, ok := messages[0].(map[string]interface{})
		if !ok {
			t.Errorf("unexpected message type: %T", messages[0])
		}
		if message["role"] != "user" {
			t.Errorf("unexpected role: %s", message["role"])
		}
		content, ok := message["content"].(string)
		if !ok {
			t.Errorf("unexpected content type: %T", message["content"])
		}
		if content != fmt.Sprintf(promptData, diff) {
			t.Errorf("unexpected content: %s", content)
		}
		w.Header().Set("Content-Type", "application/json")
		apiResponse, _ := json.Marshal(struct {
			Id      string `json:"id"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			StopReason string `json:"stop_reason"`
			Usage      struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}{
			Id: "12345",
			Content: []struct {
				Text string `json:"text"`
			}{
				{Text: `
<thinkthrough>
The changes in the provided diff appear to be related to the configuration and implementation of various worker components in a microservice application. The overall purpose of the changes seems to be:

1. Introducing more flexibility and configurability in the worker intervals and batch sizes for different workflows, such as the IPS poller, ACT workflow, initial workflow, plate type workflow, and IPS send workflow.
2. Refactoring the worker implementations to use SSM (AWS Systems Manager) parameters to dynamically override the default interval values, instead of using static intervals.
3. Ensuring consistent naming and organization of the SSM parameter keys across the different workflows.
4. Addressing a potential issue with the plate type workflow parameter name.

The changes do not appear to contain any sensitive information like API keys, passwords, or personal data. The diff also does not include any large files over 50MB that would require the use of Git LFS.

The specific modifications made in each file are as follows:

1. cfn/ssm.yaml:
   - Introduced new SSM parameters to control the batch size and interval for various workflows, such as ACT, initial, plate type, IPS send, and IPS prepare.
   - Updated the existing IPS poller parameters to use the new mapping structure.

2. internal/worker/workflowAct/worker.go:
   - Introduced a new constant IntervalKey to hold the SSM parameter key for the ACT worker interval.
   - Updated the worker creation to use the worker.NewSSMOverrideInterval function, which allows the interval to be dynamically overridden by the SSM parameter.

3. internal/worker/workflowInitial/worker.go:
   - Introduced a new constant IntervalKey to hold the SSM parameter key for the initial worker interval.
   - Updated the worker creation to use the worker.NewSSMOverrideInterval function.

4. internal/worker/workflowIps/worker.go:
   - Introduced a new constant IntervalKey to hold the SSM parameter key for the IPS worker interval.
   - Updated the worker creation to use the worker.NewSSMOverrideInterval function.

5. internal/worker/workflowIpsSend/worker.go:
   - Introduced a new constant IntervalKey to hold the SSM parameter key for the IPS send worker interval.
   - Updated the worker creation to use the worker.NewSSMOverrideInterval function.

6. internal/worker/workflowPlateType/config.go:
   - Updated the plateTypeBatchSizeKey constant to use the correct SSM parameter key.

7. internal/worker/workflowPlateType/worker.go:
   - Introduced a new constant IntervalKey to hold the SSM parameter key for the plate type worker interval.
   - Updated the worker creation to use the worker.NewSSMOverrideInterval function.

The changes seem to be aimed at improving the configurability and flexibility of the worker components, allowing the batch sizes and intervals to be adjusted based on the environment (e.g., dev, test, staging, production) without the need to modify the code directly. This approach can be beneficial for managing the application's performance and resource utilization in different environments.

The refactoring to use SSM parameters for interval configuration is a reasonable choice, as it allows the values to be easily updated without redeploying the application. The consistent naming of the SSM parameter keys across the different workflows also helps maintain code readability and maintainability.

Overall, the changes appear to be well-considered and aimed at improving the application's flexibility and configurability.
</thinkthrough>

<commit-message>
feat: Introduce dynamic worker interval configuration

This change introduces the ability to dynamically configure the worker
intervals for various workflows using AWS Systems Manager (SSM)
parameters.

Prior to this change, the worker intervals were hardcoded in the
application. This made it difficult to adjust the intervals for different
environments (e.g., dev, test, staging, production) without modifying the
code and redeploying the application.

The key changes include:

- Introduced new SSM parameters to control the batch size and interval for
  the following workflows:
  - ACT
  - Initial
  - Plate Type
  - IPS Send
  - IPS Prepare
- Updated the worker implementations to use the worker.NewSSMOverrideInterval
  function, which allows the worker interval to be dynamically overridden
  by the corresponding SSM parameter.
- Ensured consistent naming of the SSM parameter keys across the different
  workflows.
- Fixed an issue with the plate type workflow parameter name.

These changes provide more flexibility in configuring the worker
intervals and batch sizes for different environments, without the need to
modify the application code. This will help improve the application's
performance and resource utilization in various deployment scenarios.
</commit-message>

<sensitive-info-warning>
The provided diff does not contain any sensitive information like API keys, passwords, or personal data.
</sensitive-info-warning>

<large-files-warning>
The provided diff does not contain any files larger than 50MB.
</large-files-warning>
`},
			},
			StopReason: "end_turn",
			Usage: struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			}{
				InputTokens:  10,
				OutputTokens: 5,
			},
		})
		w.Write(apiResponse)
	}))
	Endpoint = ts.URL + "/v1/messages"
	t.Setenv("ANTHROPIC_API_KEY", "test-api-key")
	return ts.Close
}
