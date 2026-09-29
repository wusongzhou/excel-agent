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
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"excel-agent/utils"
	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"excel-agent/params"
)

var treeToolInfo = &schema.ToolInfo{
	Name: "tree",
	Desc: "This tool is used to view the directory tree structure; the parameter is the path to be viewed, and it returns the complete directory tree structure under that path. Directories are suffixed with a path separator; hidden files and directories (starting with '.') are skipped.",
	ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
		"path": {
			Type:     schema.String,
			Desc:     "absolute path",
			Required: true,
		},
	}),
}

func NewTreeTool(op commandline.Operator) tool.InvokableTool {
	return &tree{op: op}
}

type tree struct {
	op commandline.Operator
}

func (t *tree) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return treeToolInfo, nil
}

type treeInput struct {
	Path string `json:"path"`
}

func (t *tree) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	input := &treeInput{}

	err := json.Unmarshal([]byte(argumentsInJSON), input)
	if err != nil {
		return "", err
	}
	if len(input.Path) == 0 {
		return "path can not be empty", nil
	}
	if !filepath.IsAbs(input.Path) {
		if wd, ok := params.GetTypedContextParams[string](ctx, params.WorkDirSessionKey); ok {
			input.Path = filepath.Join(wd, input.Path)
		}
	}

	var sb strings.Builder
	walkErr := filepath.WalkDir(input.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			sb.WriteString(fmt.Sprintf("[error] %v: %v\n", path, err))
			return nil
		}
		if path != input.Path && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			sb.WriteString(path + string(filepath.Separator) + "\n")
		} else {
			sb.WriteString(path + "\n")
		}
		return nil
	})
	if walkErr != nil {
		return fmt.Sprintf("tree error: %v, path: %v", walkErr, input.Path), nil
	}
	return utils.TruncateString(sb.String(), maxToolOutputChars), nil
}
