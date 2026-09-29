/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"excel-agent/params"
	"github.com/cloudwego/eino-ext/components/tool/commandline"
)

type LocalOperator struct{}

func (l *LocalOperator) ReadFile(ctx context.Context, path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (l *LocalOperator) WriteFile(ctx context.Context, path, content string) error {
	return os.WriteFile(path, []byte(content), 0o666)
}

func (l *LocalOperator) IsDirectory(ctx context.Context, path string) (bool, error) {
	return true, nil
}

func (l *LocalOperator) Exists(ctx context.Context, path string) (bool, error) {
	return true, nil
}

func (l *LocalOperator) RunCommand(ctx context.Context, command []string) (*commandline.CommandOutput, error) {
	wd, ok := params.GetTypedContextParams[string](ctx, params.WorkDirSessionKey)
	if !ok {
		return nil, fmt.Errorf("work dir not found")
	}

	var shellCmd []string
	switch runtime.GOOS {
	case "windows":
		shellCmd = buildWindowsCommand(command)
	default:
		shellCmd = []string{"/bin/sh", "-c", strings.Join(command, " ")}
	}

	cmd := exec.CommandContext(ctx, shellCmd[0], shellCmd[1:]...)
	cmd.Dir = wd
	// Python defaults to the locale encoding (GBK on zh-CN Windows) for piped output; force UTF-8 so Chinese output stays readable.
	cmd.Env = append(os.Environ(), "PYTHONUTF8=1", "PYTHONIOENCODING=utf-8")

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.Stdout = outBuf
	cmd.Stderr = errBuf
	err := cmd.Run()
	if err != nil {
		err = fmt.Errorf("internal error:\ncommand: %v\n\nerr: %v\n\nexec error: %v", cmd.String(), err, errBuf.String())
		return nil, err
	}
	return &commandline.CommandOutput{
		Stdout: outBuf.String(),
		Stderr: errBuf.String(),
	}, nil
}

// buildWindowsCommand assembles the command line for Windows. PowerShell is
// the default (pwsh if installed, otherwise Windows PowerShell) because it
// offers the agent a richer command set than cmd.exe; setting
// EXCEL_AGENT_WINDOWS_SHELL=cmd restores the previous behaviour.
func buildWindowsCommand(command []string) []string {
	if os.Getenv("EXCEL_AGENT_WINDOWS_SHELL") == "cmd" {
		return append([]string{"cmd.exe", "/C"}, command...)
	}

	shell := "powershell"
	if path, err := exec.LookPath("pwsh"); err == nil {
		shell = path
	}

	var script string
	if len(command) == 1 {
		script = command[0]
	} else {
		// A program and its arguments (e.g. python_runner); the call
		// operator (&) keeps quoted paths intact.
		quoted := make([]string, len(command))
		for i, arg := range command {
			quoted[i] = "'" + strings.ReplaceAll(arg, "'", "''") + "'"
		}
		script = "& " + strings.Join(quoted, " ")
	}

	// Windows PowerShell exits 0 even when the last native command failed and
	// emits output in the OEM code page (GBK on zh-CN); both are fixed here.
	script = "[Console]::OutputEncoding = [System.Text.Encoding]::UTF8\n" +
		script +
		"\nif ($LASTEXITCODE) { exit $LASTEXITCODE } elseif (-not $?) { exit 1 }"

	return []string{shell, "-NoProfile", "-NonInteractive", "-Command", script}
}
