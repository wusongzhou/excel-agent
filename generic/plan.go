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

package generic

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/schema"

	"excel-agent/utils"
)

type Step struct {
	Index int    `json:"index"`
	Desc  string `json:"desc"`
}

func (s *Step) UnmarshalJSON(data []byte) error {
	type Alias struct {
		Index       int    `json:"index"`
		Desc        string `json:"desc"`
		Instruction string `json:"instruction"`
	}
	alias := &Alias{}
	if err := json.Unmarshal(data, alias); err != nil {
		return err
	}
	s.Index = alias.Index
	s.Desc = alias.Desc
	if s.Desc == "" {
		s.Desc = alias.Instruction
	}
	return nil
}

type Plan struct {
	Steps []Step `json:"steps"`
}

func (p *Plan) FirstStep() string {
	if len(p.Steps) == 0 {
		return ""
	}
	b, _ := json.Marshal(p.Steps[0])
	return string(b)
}

func (p *Plan) MarshalJSON() ([]byte, error) {
	type Alias Plan
	return json.Marshal((*Alias)(p))
}

// UnmarshalJSON is tolerant of the wrappers OpenAI-protocol compatible models
// add around plan output (markdown code fences, leading/trailing prose,
// slightly malformed JSON), because not every service honours the
// response_format=json_schema parameter.
func (p *Plan) UnmarshalJSON(bytes []byte) error {
	type Alias Plan
	a := (*Alias)(p)

	s := extractPlanJSON(string(bytes))
	if err := json.Unmarshal([]byte(s), a); err == nil {
		return nil
	}
	return json.Unmarshal([]byte(utils.RepairJSON(s)), a)
}

var jsonFencePattern = regexp.MustCompile(`(?s)^\s*` + "```" + `(?:json)?\s*(.*?)` + "```" + `\s*$`)

func extractPlanJSON(s string) string {
	s = strings.TrimSpace(s)
	if m := jsonFencePattern.FindStringSubmatch(s); len(m) > 1 {
		s = strings.TrimSpace(m[1])
	}
	if i := strings.Index(s, "{"); i >= 0 {
		if j := strings.LastIndex(s, "}"); j > i {
			return s[i : j+1]
		}
	}
	return s
}

var PlanToolInfo = &schema.ToolInfo{
	Name: "create_plan",
	Desc: "Generates a structured, step-by-step execution plan to solve a given complex task. Each step in the plan must be assigned to a specialized agent and must have a clear, actionable description.",
	ParamsOneOf: schema.NewParamsOneOfByParams(
		map[string]*schema.ParameterInfo{
			"steps": {
				Type: schema.Array,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.Object,
					SubParams: map[string]*schema.ParameterInfo{
						"index": {
							Type:     schema.Integer,
							Desc:     "The sequential number of this step in the overall plan. **Must start from 1 and increment by exactly 1 for each subsequent step.**",
							Required: true,
						},
						"desc": {
							Type:     schema.String,
							Desc:     "A clear, concise, and actionable description of the specific task to be performed in this step. It should be a direct instruction for the assigned agent.",
							Required: true,
						},
					},
				},
				Desc:     "different steps to follow, should be in sorted order",
				Required: true,
			},
		},
	),
}
