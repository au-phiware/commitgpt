# CommitGPT

CommitGPT is a tool that generates commit messages using Claude, an AI model by
Anthropic. It aims to improve the consistency and quality of git commit
messages by integrating AI assistance into the development workflow.

## Prerequisites

- Go 1.x
- Git
- Pandoc
- libsecret (for storing the API key securely)

## Installation

### Using Nix

If you have Nix installed, you can build and install CommitGPT using the following commands:

```sh
nix-build
nix-env -i ./result
```

This will build the project and install the binary in your Nix profile. The
binary is wrapped in a shell script that sets up the necessary environment
variables, so you can use it directly as a prepare-commit-msg git-hook (see
[Usage](#usage)).

### Using Go

To install CommitGPT using Go, follow these steps:

1. Clone the repository:
   ```sh
   git clone https://github.com/au-phiware/commitgpt.git
   cd commitgpt
   ```

2. Build the project:
   ```sh
   go build -o commitgpt
   ```

3. Install the binary:
   ```sh
   go install
   ```

## Configuration

Before using CommitGPT, you need to set up your Anthropic [API
key](https://console.anthropic.com/settings/keys):

1. Store your API key securely using `secret-tool`:
   ```sh
   secret-tool store --label="Anthropic API Key" anthropic-api-key commitgpt
   ```

2. Create a directory for logs:
   ```sh
   mkdir -p ~/.config/anthropic/logs
   ```

## Usage

To use CommitGPT as a Git hook for preparing commit messages:

1. Create a file named `prepare-commit-msg` in your Git repository's
   `.git/hooks/` directory with the following content:
   ```sh
   #!/bin/sh
   export ANTHROPIC_API_KEY=$(secret-tool lookup anthropic-api-key commitgpt)
   export ANTHROPIC_LOG_DIR=~/.config/anthropic/logs
   commitgpt $1 $2 $3
   ```

2. Make the hook executable:
   ```sh
   chmod +x .git/hooks/prepare-commit-msg
   ```

Now, when you make a commit, CommitGPT will automatically generate a commit
message based on your changes.

## License

This project is licensed under the MIT License.

## Contributing

Contributions are welcome! Please feel free to submit a [Pull
Request](https://github.com/au-phiware/commitgpt).

## Acknowledgements

This project uses the Claude API by Anthropic to generate commit messages.

The metaprompting technique used in this project is based on the Metaprompt
Workbook by Anthropic. You can find the workbook here: [Metaprompt
Workbook](https://colab.research.google.com/drive/1SoAajN8CBYTl79VyTwxtxncfCWlHlyy9#scrollTo=NTOiFKNxqoq2).
We highly recommend checking it out to understand the underlying methodology of
our commit message generation process.
