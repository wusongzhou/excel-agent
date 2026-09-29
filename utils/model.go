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

package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	arkmodel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

const defaultZhipuBaseURL = "https://open.bigmodel.cn/api/paas/v4"

type CreateChatModelOption func(o *option)

// NewChatModel builds the chat model from the first configured provider:
// ARK_* first, then ZHIPU_*, then OPENAI_*. Configuration is read from
// environment variables, which may come from the project .env file.
func NewChatModel(ctx context.Context, opts ...CreateChatModelOption) (cm model.ToolCallingChatModel, err error) {
	o := &option{}
	for _, opt := range opts {
		opt(o)
	}

	if modelName := os.Getenv("ARK_MODEL"); modelName != "" {
		conf := &ark.ChatModelConfig{
			APIKey:      os.Getenv("ARK_API_KEY"),
			BaseURL:     os.Getenv("ARK_BASE_URL"),
			Region:      os.Getenv("ARK_REGION"),
			Model:       modelName,
			MaxTokens:   o.MaxTokens,
			Temperature: o.Temperature,
			TopP:        o.TopP,
		}
		if o.DisableThinking != nil && *o.DisableThinking {
			conf.Thinking = &arkmodel.Thinking{
				Type: arkmodel.ThinkingTypeDisabled,
			}
		}
		if o.JsonSchema != nil {
			conf.ResponseFormat = &ark.ResponseFormat{
				Type: arkmodel.ResponseFormatJSONSchema,
				JSONSchema: &arkmodel.ResponseFormatJSONSchemaJSONSchemaParam{
					Name:        o.JsonSchema.Name,
					Description: o.JsonSchema.Description,
					Schema:      o.JsonSchema.JSONSchema,
					Strict:      o.JsonSchema.Strict,
				},
			}
		}
		cm, err = ark.NewChatModel(ctx, conf)

	} else if modelName := os.Getenv("ZHIPU_MODEL"); modelName != "" {
		cm, err = newOpenAICompatModel(ctx, &openai.ChatModelConfig{
			APIKey:          os.Getenv("ZHIPU_API_KEY"),
			BaseURL:         zhipuBaseURL(),
			Model:           modelName,
			MaxTokens:       o.MaxTokens,
			Temperature:     o.Temperature,
			TopP:            o.TopP,
			ReasoningEffort: reasoningEffort(),
		}, o, true)

	} else if modelName := os.Getenv("OPENAI_MODEL"); modelName != "" {
		cm, err = newOpenAICompatModel(ctx, &openai.ChatModelConfig{
			APIKey: os.Getenv("OPENAI_API_KEY"),
			ByAzure: func() bool {
				return os.Getenv("OPENAI_BY_AZURE") == "true"
			}(),
			BaseURL:         os.Getenv("OPENAI_BASE_URL"),
			Model:           modelName,
			MaxTokens:       o.MaxTokens,
			Temperature:     o.Temperature,
			TopP:            o.TopP,
			ReasoningEffort: reasoningEffort(),
		}, o, false)
	}
	if err != nil {
		return nil, err
	}
	if cm == nil {
		return nil, fmt.Errorf("no chat model configured: set ARK_MODEL, ZHIPU_MODEL or OPENAI_MODEL via environment variables or the project .env file")
	}

	return cm, nil
}

// newOpenAICompatModel builds an OpenAI-protocol client shared by the
// ZHIPU_* and OPENAI_* provider configs. On bigmodel the strict
// response_format=json_schema parameter is accepted but silently ignored for
// glm-5.3-flash, so zhipu requests the honoured json_object mode instead and
// leaves the output shape to the prompt and the tolerant plan parsing.
func newOpenAICompatModel(ctx context.Context, conf *openai.ChatModelConfig, o *option, zhipu bool) (model.ToolCallingChatModel, error) {
	if o.JsonSchema != nil && !jsonSchemaDisabled() {
		if zhipu {
			conf.ResponseFormat = &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			}
		} else {
			conf.ResponseFormat = &openai.ChatCompletionResponseFormat{
				Type:       openai.ChatCompletionResponseFormatTypeJSONSchema,
				JSONSchema: o.JsonSchema,
			}
		}
	}
	return openai.NewChatModel(ctx, conf)
}

func zhipuBaseURL() string {
	if u := os.Getenv("ZHIPU_BASE_URL"); u != "" {
		return u
	}
	return defaultZhipuBaseURL
}

// reasoningEffort maps EXCEL_AGENT_REASONING_EFFORT to the reasoning_effort
// request parameter (e.g. low/medium/high/max); empty means the provider
// default. GLM-5.3-Flash cannot disable thinking, so lowering the effort is
// the main speed lever.
func reasoningEffort() openai.ReasoningEffortLevel {
	return openai.ReasoningEffortLevel(os.Getenv("EXCEL_AGENT_REASONING_EFFORT"))
}

// NewVisionModel resolves the model behind the image_reader tool: the Ark
// vision config first, then an explicit ZHIPU_VISION_MODEL, and finally the
// configured Zhipu main model, which is a VLM (e.g. glm-5.3-flash). Returns
// nil when no vision capability is configured, and the image_reader tool is
// then simply not registered.
func NewVisionModel(ctx context.Context) (model.BaseChatModel, error) {
	if name := os.Getenv("ARK_VISION_MODEL"); name != "" {
		return ark.NewChatModel(ctx, &ark.ChatModelConfig{
			APIKey:  os.Getenv("ARK_VISION_API_KEY"),
			BaseURL: os.Getenv("ARK_VISION_BASE_URL"),
			Region:  os.Getenv("ARK_VISION_REGION"),
			Model:   name,
		})
	}

	if name := os.Getenv("ZHIPU_VISION_MODEL"); name != "" {
		apiKey := os.Getenv("ZHIPU_VISION_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("ZHIPU_API_KEY")
		}
		baseURL := zhipuBaseURL()
		if u := os.Getenv("ZHIPU_VISION_BASE_URL"); u != "" {
			baseURL = u
		}
		return openai.NewChatModel(ctx, &openai.ChatModelConfig{
			APIKey:  apiKey,
			BaseURL: baseURL,
			Model:   name,
		})
	}

	if os.Getenv("ZHIPU_MODEL") != "" {
		return NewChatModel(ctx)
	}

	return nil, nil
}

type option struct {
	MaxTokens       *int
	Temperature     *float32
	TopP            *float32
	DisableThinking *bool
	JsonSchema      *openai.ChatCompletionResponseFormatJSONSchema
}

func WithMaxTokens(maxTokens int) CreateChatModelOption {
	return func(o *option) {
		o.MaxTokens = &maxTokens
	}
}

func WithTemperature(temp float32) CreateChatModelOption {
	return func(o *option) {
		o.Temperature = &temp
	}
}

func WithTopP(topP float32) CreateChatModelOption {
	return func(o *option) {
		o.TopP = &topP
	}
}

func WithDisableThinking(disable bool) CreateChatModelOption {
	return func(o *option) {
		o.DisableThinking = &disable
	}
}

func WithResponseFormatJsonSchema(schema *openai.ChatCompletionResponseFormatJSONSchema) CreateChatModelOption {
	return func(o *option) {
		o.JsonSchema = schema
	}
}

// jsonSchemaDisabled reports whether strict json_schema responses should be
// skipped: not every OpenAI-protocol compatible service supports the
// response_format=json_schema parameter, and it makes some of them fail.
// Set OPENAI_DISABLE_JSON_SCHEMA=true to opt out.
func jsonSchemaDisabled() bool {
	v := os.Getenv("OPENAI_DISABLE_JSON_SCHEMA")
	return v == "true" || v == "1"
}
