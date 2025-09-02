// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package prompt

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/valyala/fasttemplate"

	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/consts"
	"github.com/coze-dev/cozeloop-go/internal/httpclient"
	"github.com/coze-dev/cozeloop-go/internal/logger"
	"github.com/coze-dev/cozeloop-go/internal/trace"
	"github.com/coze-dev/cozeloop-go/internal/util"
	"github.com/coze-dev/cozeloop-go/spec/tracespec"
)

type Provider struct {
	openAPIClient *OpenAPIClient
	traceProvider *trace.Provider
	cache         *PromptCache
	config        Options
}

type Options struct {
	WorkspaceID                string
	PromptCacheMaxCount        int
	PromptCacheRefreshInterval time.Duration
	PromptTrace                bool
}

type GetPromptParam struct {
	PromptKey string
	Version   string
	Label     string
}

type GetPromptOptions struct {
}

type PromptFormatOptions struct {
}

func NewPromptProvider(httpClient *httpclient.Client, traceProvider *trace.Provider, options Options) *Provider {
	openAPI := &OpenAPIClient{httpClient: httpClient}
	cache := newPromptCache(options.WorkspaceID, openAPI,
		withAsyncUpdate(true),
		withUpdateInterval(options.PromptCacheRefreshInterval),
		withMaxCacheSize(options.PromptCacheMaxCount))
	return &Provider{
		openAPIClient: openAPI,
		traceProvider: traceProvider,
		cache:         cache,
		config:        options,
	}
}

func (p *Provider) GetPrompt(ctx context.Context, param GetPromptParam, options GetPromptOptions) (prompt *entity.Prompt, err error) {
	if p.config.PromptTrace && p.traceProvider != nil {
		var promptHubSpan *trace.Span
		var spanErr error
		ctx, promptHubSpan, spanErr = p.traceProvider.StartSpan(ctx, consts.TracePromptHubSpanName, tracespec.VPromptHubSpanType,
			trace.StartSpanOptions{Scene: tracespec.VScenePromptHub})
		if spanErr != nil {
			logger.CtxWarnf(ctx, "start prompt hub span failed: %v", err)
		}
		defer func() {
			if promptHubSpan != nil {
				promptHubSpan.SetTags(ctx, map[string]any{
					tracespec.PromptKey: param.PromptKey,
					tracespec.Input: util.ToJSON(map[string]any{
						tracespec.PromptKey:     param.PromptKey,
						tracespec.PromptVersion: param.Version,
						tracespec.PromptLabel:   param.Label,
					}),
				})
				if prompt != nil {
					promptHubSpan.SetTags(ctx, map[string]any{
						tracespec.PromptVersion: prompt.Version, // actual version
						tracespec.Output:        util.ToJSON(prompt),
					})
				}
				if err != nil {
					promptHubSpan.SetStatusCode(ctx, util.GetErrorCode(err))
					promptHubSpan.SetError(ctx, err)
				}
				promptHubSpan.Finish(ctx)
			}
		}()
	}
	return p.doGetPrompt(ctx, param, options)
}

func (p *Provider) doGetPrompt(ctx context.Context, param GetPromptParam, options GetPromptOptions) (prompt *entity.Prompt, err error) {
	defer func() {
		// object cache item should be read only
		prompt = prompt.DeepCopy()
	}()
	// Get from cache
	if cached, ok := p.cache.Get(param.PromptKey, param.Version, param.Label); ok {
		return cached, nil
	}

	// Cache miss, fetch from server
	promptResults, err := p.openAPIClient.MPullPrompt(ctx, MPullPromptRequest{
		WorkSpaceID: p.config.WorkspaceID,
		Queries: []PromptQuery{
			{
				PromptKey: param.PromptKey,
				Version:   param.Version,
				Label:     param.Label,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(promptResults) == 0 || promptResults[0].Prompt == nil {
		return nil, nil
	}

	// Cache the result
	result := toModelPrompt(promptResults[0].Prompt)
	p.cache.Set(promptResults[0].Query.PromptKey, promptResults[0].Query.Version, promptResults[0].Query.Label, result)

	return result, nil
}

func (p *Provider) PromptFormat(ctx context.Context, prompt *entity.Prompt, variables map[string]any, options PromptFormatOptions) (messages []*entity.Message, err error) {
	if prompt == nil || prompt.PromptTemplate == nil {
		return nil, nil
	}
	if p.config.PromptTrace && p.traceProvider != nil {
		var promptTemplateSpan *trace.Span
		var spanErr error
		ctx, promptTemplateSpan, spanErr = p.traceProvider.StartSpan(ctx, consts.TracePromptTemplateSpanName, tracespec.VPromptTemplateSpanType,
			trace.StartSpanOptions{Scene: tracespec.VScenePromptTemplate})
		if spanErr != nil {
			logger.CtxWarnf(ctx, "start prompt template span failed: %v", err)
		}
		defer func() {
			if promptTemplateSpan != nil {
				promptTemplateSpan.SetTags(ctx, map[string]any{
					tracespec.PromptKey:     prompt.PromptKey,
					tracespec.PromptVersion: prompt.Version,
					tracespec.Input:         util.ToJSON(toSpanPromptInput(prompt.PromptTemplate.Messages, variables)),
					tracespec.Output:        util.ToJSON(toSpanMessages(messages)),
				})
				if err != nil {
					promptTemplateSpan.SetStatusCode(ctx, util.GetErrorCode(err))
					promptTemplateSpan.SetError(ctx, err)
				}
				promptTemplateSpan.Finish(ctx)
			}
		}()
	}
	return p.doPromptFormat(ctx, prompt.DeepCopy(), variables)
}

func (p *Provider) doPromptFormat(ctx context.Context, prompt *entity.Prompt, variables map[string]any) (results []*entity.Message, err error) {
	if prompt.PromptTemplate == nil || len(prompt.PromptTemplate.Messages) == 0 {
		return nil, nil
	}
	// validate variable value type
	err = validateVariableValuesType(prompt.PromptTemplate.VariableDefs, variables)
	if err != nil {
		return nil, err
	}
	results, err = formatNormalMessages(prompt.PromptTemplate.TemplateType, prompt.PromptTemplate.Messages, prompt.PromptTemplate.VariableDefs, variables)
	if err != nil {
		return nil, err
	}
	results, err = formatPlaceholderMessages(results, variables)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func validateVariableValuesType(variableDefs []*entity.VariableDef, variables map[string]any) error {
	for _, variableDef := range variableDefs {
		if variableDef == nil {
			continue
		}
		val := variables[variableDef.Key]
		if val == nil {
			continue
		}
		switch variableDef.Type {
		case entity.VariableTypeString:
			if _, ok := val.(string); !ok {
				return consts.ErrInvalidParam.Wrap(fmt.Errorf("type of variable '%s' should be string", variableDef.Key))
			}
		case entity.VariableTypePlaceholder:
			switch val.(type) {
			case []*entity.Message, []entity.Message, *entity.Message, entity.Message:
				return nil
			default:
				return consts.ErrInvalidParam.Wrap(fmt.Errorf("type of variable '%s' should be Message like object", variableDef.Key))
			}
		case entity.VariableTypeBoolean:
			if _, ok := val.(bool); !ok {
				return consts.ErrInvalidParam.Wrap(fmt.Errorf("type of variable '%s' should be bool", variableDef.Key))
			}
		case entity.VariableTypeInteger:
			if _, ok := val.(int); ok {
				return nil
			}
			if _, ok := val.(int64); ok {
				return nil
			}
			if _, ok := val.(int32); ok {
				return nil
			}
			return consts.ErrInvalidParam.Wrap(fmt.Errorf("type of variable '%s' should be int or int64 or int32", variableDef.Key))
		case entity.VariableTypeFloat:
			if _, ok := val.(float64); ok {
				return nil
			}
			if _, ok := val.(float32); ok {
				return nil
			}
			return consts.ErrInvalidParam.Wrap(fmt.Errorf("type of variable '%s' should be float64 or float32", variableDef.Key))
		case entity.VariableTypeArrayString:
			if _, ok := val.([]string); !ok {
				return consts.ErrInvalidParam.Wrap(fmt.Errorf("type of variable '%s' should be []string", variableDef.Key))
			}
		case entity.VariableTypeArrayBoolean:
			if _, ok := val.([]bool); !ok {
				return consts.ErrInvalidParam.Wrap(fmt.Errorf("type of variable '%s' should be []bool", variableDef.Key))
			}
		case entity.VariableTypeArrayInteger:
			if _, ok := val.([]int); ok {
				return nil
			}
			if _, ok := val.([]int64); ok {
				return nil
			}
			if _, ok := val.([]int32); ok {
				return nil
			}
			return consts.ErrInvalidParam.Wrap(fmt.Errorf("type of variable '%s' should be []int or []int64 or []int32", variableDef.Key))
		case entity.VariableTypeArrayFloat:
			if _, ok := val.([]float64); ok {
				return nil
			}
			if _, ok := val.([]float32); ok {
				return nil
			}
			return consts.ErrInvalidParam.Wrap(fmt.Errorf("type of variable '%s' should be []float64 or []float32", variableDef.Key))
		case entity.VariableTypeMultiPart:
			if _, ok := val.([]*entity.ContentPart); !ok {
				return consts.ErrInvalidParam.Wrap(fmt.Errorf("type of variable '%s' should be multi_part", variableDef.Key))
			}
		}
	}
	return nil
}

func formatNormalMessages(templateType entity.TemplateType,
	messages []*entity.Message,
	variableDefs []*entity.VariableDef,
	variableVals map[string]any) (results []*entity.Message, err error) {
	variableDefMap := make(map[string]*entity.VariableDef)
	for _, variableDef := range variableDefs {
		if variableDef != nil {
			variableDefMap[variableDef.Key] = variableDef
		}
	}
	for _, message := range messages {
		if message == nil {
			continue
		}
		// placeholder is not processed here
		if message.Role == entity.RolePlaceholder {
			results = append(results, message)
			continue
		}
		// render content
		if util.PtrValue(message.Content) != "" {
			renderedContent, err := renderTextContent(templateType, util.PtrValue(message.Content), variableDefMap, variableVals)
			if err != nil {
				return nil, err
			}
			message.Content = util.Ptr(renderedContent)
		}
		// render parts
		message.Parts = formatMultiPart(templateType, message.Parts, variableDefMap, variableVals)
		results = append(results, message)
	}
	return results, nil
}

func formatMultiPart(templateType entity.TemplateType,
	parts []*entity.ContentPart,
	defMap map[string]*entity.VariableDef,
	valMap map[string]any) []*entity.ContentPart {
	var formatedParts []*entity.ContentPart
	// render text
	for _, part := range parts {
		if part == nil {
			continue
		}
		if part.Type == entity.ContentTypeText && util.PtrValue(part.Text) != "" {
			renderedText, err := renderTextContent(templateType, util.PtrValue(part.Text), defMap, valMap)
			if err != nil {
				return nil
			}
			part.Text = util.Ptr(renderedText)
		}
	}
	// render multipart variable
	for _, part := range parts {
		if part == nil {
			continue
		}
		if part.Type == entity.ContentTypeMultiPartVariable && util.PtrValue(part.Text) != "" {
			multiPartVariableKey := util.PtrValue(part.Text)
			if vardef, ok := defMap[multiPartVariableKey]; ok {
				if value, ok := valMap[multiPartVariableKey]; ok {
					if vardef != nil && value != nil && vardef.Type == entity.VariableTypeMultiPart {
						if multiPartValues, ok := value.([]*entity.ContentPart); ok {
							formatedParts = append(formatedParts, multiPartValues...)
						}
					}
				}
			}
		} else {
			formatedParts = append(formatedParts, part)
		}
	}
	// fileter
	var filtered []*entity.ContentPart
	for _, pt := range formatedParts {
		if pt == nil {
			continue
		}
		if util.PtrValue(pt.Text) != "" || pt.ImageURL != nil {
			filtered = append(filtered, pt)
		}
	}
	return filtered
}

func formatPlaceholderMessages(messages []*entity.Message, variableVals map[string]any) (results []*entity.Message, err error) {
	expandedMessages := make([]*entity.Message, 0)
	for _, message := range messages {
		if message != nil && message.Role == entity.RolePlaceholder {
			placeholderVariableName := util.PtrValue(message.Content)
			if placeholderVariable, ok := variableVals[placeholderVariableName]; ok && placeholderVariable != nil {
				placeholderMessages, err := convertMessageLikeObjectToMessages(placeholderVariable)
				if err != nil {
					return nil, err
				}
				expandedMessages = append(expandedMessages, placeholderMessages...)
			}
		} else {
			expandedMessages = append(expandedMessages, message)
		}
	}
	return expandedMessages, nil
}

func renderTextContent(templateType entity.TemplateType,
	templateStr string,
	variableDefMap map[string]*entity.VariableDef,
	variableVals map[string]any) (string, error) {
	switch templateType {
	case entity.TemplateTypeNormal:
		return fasttemplate.ExecuteFuncString(templateStr, consts.PromptNormalTemplateStartTag, consts.PromptNormalTemplateEndTag, func(w io.Writer, tag string) (int, error) {
			// If not in variable definition, don't replace and return directly
			if variableDefMap[tag] == nil {
				return w.Write([]byte(consts.PromptNormalTemplateStartTag + tag + consts.PromptNormalTemplateEndTag))
			}
			// Otherwise replace
			if val, ok := variableVals[tag]; ok {
				return w.Write([]byte(fmt.Sprint(val)))
			}
			return 0, nil
		}), nil
	case entity.TemplateTypeJinja2:
		return util.InterpolateJinja2(templateStr, variableVals)
	default:
		return "", consts.ErrInternal.Wrap(fmt.Errorf("unknown template type: %s", templateType))
	}
}

func convertMessageLikeObjectToMessages(object any) (messages []*entity.Message, err error) {
	switch object.(type) {
	case []*entity.Message:
		return object.([]*entity.Message), nil
	case []entity.Message:
		for _, message := range object.([]entity.Message) {
			messages = append(messages, &message)
		}
		return messages, nil
	case *entity.Message:
		return []*entity.Message{object.(*entity.Message)}, nil
	case entity.Message:
		message := object.(entity.Message)
		return []*entity.Message{&message}, nil
	default:
		return nil, consts.ErrInvalidParam.Wrap(fmt.Errorf("placeholder message variable is invalid"))
	}
}
