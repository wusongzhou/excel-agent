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

package tools

import (
	"context"
	"encoding/json"
	"runtime"
	"strings"

	"excel-agent/utils"
	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

var bashToolParams = schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
	"command": {
		Type:     "string",
		Desc:     "The command to execute",
		Required: true,
	},
})

func bashToolDesc() string {
	if runtime.GOOS == "windows" {
		return `Run commands in the Windows command prompt (cmd.exe).
* When invoking this tool, the contents of the "command" parameter does NOT need to be XML-escaped.
* Commands are executed via 'cmd.exe /C' in the task working directory, so use Windows command syntax (dir, type, copy, del, findstr...). Unix tools such as rm, ls, grep and sed are NOT available.
* Python is available; prefer the dedicated python_runner tool for running Python code.
* Each command runs in a fresh process, so state such as 'cd' or environment variables does NOT persist between calls.
* Please avoid commands that may produce a very large amount of output.`
	}
	return `Run commands in a bash shell
* When invoking this tool, the contents of the "command" parameter does NOT need to be XML-escaped.
* You don't have access to the internet via this tool.
* You do have access to a mirror of common linux and python packages via apt and pip.
* Each command runs in a fresh process, so state such as 'cd' or environment variables does NOT persist between calls.
* To inspect a particular line range of a file, e.g. lines 10-25, try 'sed -n 10,25p /path/to/the/file'.
* Please avoid commands that may produce a very large amount of output.
* Please run long lived commands in the background, e.g. 'sleep 10 &' or start a server in the background.`
}

func NewBashTool(op commandline.Operator) tool.InvokableTool {
	return &bashTool{op: op}
}

type bashTool struct {
	op commandline.Operator
}

func (b *bashTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "bash",
		Desc:        bashToolDesc(),
		ParamsOneOf: bashToolParams,
	}, nil
}

type shellInput struct {
	Command string `json:"command"`
}

func (b *bashTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	input := &shellInput{}
	err := json.Unmarshal([]byte(argumentsInJSON), input)
	if err != nil {
		return "", err
	}
	if len(input.Command) == 0 {
		return "command cannot be empty", nil
	}
	o := tool.GetImplSpecificOptions(&options{b.op}, opts...)
	cmd, err := o.op.RunCommand(ctx, []string{input.Command})
	if err != nil {
		if strings.HasPrefix(err.Error(), "internal error") {
			return err.Error(), nil
		}
		return "", err
	}
	return utils.FormatCommandOutput(cmd), nil
}
