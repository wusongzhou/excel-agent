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
	"path/filepath"
	"strings"

	"excel-agent/utils"
	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"excel-agent/generic"
	"excel-agent/params"
)

var readFileToolInfo = &schema.ToolInfo{
	Name: "read_file",
	Desc: `This tool is used for reading file content, with parameters including the file path, starting line, and the number of lines to read. Content will be truncated if it is too long.
For xlsx and xlsm files, each sheet's information will be returned sequentially upon a single call. If multiple sheets' information is needed, only one call is required. The call will return the headers, merged cell information, and the first n_rows of data for each sheet.`,
	ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
		"path": {
			Type:     schema.String,
			Desc:     "file absolute path",
			Required: true,
		},
		"start_row": {
			Type: schema.Integer,
			Desc: "The starting line defaults to 1, meaning reading begins from the first line. Not applied to excel files.",
		},
		"n_rows": {
			Type: schema.Integer,
			Desc: "Number of rows to read, -1 means reading from start_row to the end of the file, default is 20 rows. For xlsx and xlsm files, the default is 10 rows.",
		},
	}),
}

const (
	defaultReadRows      = 20
	defaultExcelReadRows = 10
	// maxExcelReadRows caps the "read to the end" case for excel files; the
	// response is truncated anyway, so unbounded sheet iteration is pointless.
	maxExcelReadRows = 10000
)

func NewReadFileTool(op commandline.Operator) tool.InvokableTool {
	return &readFile{op: op}
}

type readFile struct {
	op commandline.Operator
}

func (r *readFile) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return readFileToolInfo, nil
}

type readFileInput struct {
	Path     string `json:"path"`
	StartRow int    `json:"start_row"`
	NRows    int    `json:"n_rows"`
}

func (r *readFile) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	input := &readFileInput{}
	err := json.Unmarshal([]byte(argumentsInJSON), input)
	if err != nil {
		return "", err
	}
	if input.Path == "" {
		return "path can not be empty", nil
	}
	if !filepath.IsAbs(input.Path) {
		if wd, ok := params.GetTypedContextParams[string](ctx, params.WorkDirSessionKey); ok {
			input.Path = filepath.Join(wd, input.Path)
		}
	}
	if input.StartRow <= 0 {
		input.StartRow = 1
	}

	switch strings.ToLower(filepath.Ext(input.Path)) {
	case ".xlsx", ".xlsm":
		return r.readExcel(input)
	default:
		return r.readText(ctx, input)
	}
}

func (r *readFile) readText(ctx context.Context, input *readFileInput) (string, error) {
	content, err := r.op.ReadFile(ctx, input.Path)
	if err != nil {
		return fmt.Sprintf("read file error: %v, file path: %v", err, input.Path), nil
	}

	lines := strings.Split(content, "\n")
	nRows := input.NRows
	if nRows == 0 {
		nRows = defaultReadRows
	}
	lo := input.StartRow - 1
	if lo > len(lines) {
		lo = len(lines)
	}
	hi := len(lines)
	if nRows > 0 && lo+nRows < hi {
		hi = lo + nRows
	}
	return utils.TruncateString(strings.Join(lines[lo:hi], "\n"), maxToolOutputChars), nil
}

func (r *readFile) readExcel(input *readFileInput) (string, error) {
	maxRows := input.NRows
	switch {
	case maxRows < 0:
		maxRows = maxExcelReadRows
	case maxRows == 0:
		maxRows = defaultExcelReadRows
	}

	pf, err := generic.PreviewExcelDocument(input.Path, maxRows)
	if err != nil {
		return fmt.Sprintf("read file error: %v, file path: %v", err, input.Path), nil
	}
	return utils.TruncateString(utils.ToJSONString(pf), maxToolOutputChars), nil
}
