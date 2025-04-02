// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: MIT

package prompt

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/valyala/fasttemplate"

	attribute "code.byted.org/flowdevops/loop-go/attribute/trace"
	"github.com/coze-dev/cozeloop-go/entity"
	"github.com/coze-dev/cozeloop-go/internal/consts"
	"github.com/coze-dev/cozeloop-go/internal/httpclient"
	"github.com/coze-dev/cozeloop-go/internal/logger"
	"github.com/coze-dev/cozeloop-go/internal/trace"
	"github.com/coze-dev/cozeloop-go/internal/util"
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
}

type GetPromptOptions struct {
}

type PromptFormatOptions struct {
}

func NewPromptProvider(httpClient *httpclient.Client, traceProvider *trace.Provider, options Options) *Provider {
	openAPI := &OpenAPIClient{httpClient: httpClient}
	cache := newPromptCache(options.WorkspaceID, openAPI, withAsyncUpdate(true), withUpdateInterval(options.PromptCacheRefreshInterval))
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
		ctx, promptHubSpan, spanErr = p.traceProvider.StartSpan(ctx, consts.TracePromptHubSpanName, attribute.VPromptHubSpanType,
			trace.StartSpanOptions{Scene: attribute.VScenePromptHub})
		if spanErr != nil {
			logger.CtxWarnf(ctx, "start prompt hub span failed: %v", err)
		}
		defer func() {
			if promptHubSpan != nil {
				promptHubSpan.SetTags(ctx, map[string]any{
					attribute.PromptKey: param.PromptKey,
					attribute.Input: util.ToJSON(map[string]any{
						attribute.PromptKey:     param.PromptKey,
						attribute.PromptVersion: param.Version,
					}),
				})
				if prompt != nil {
					promptHubSpan.SetTags(ctx, map[string]any{
						attribute.PromptVersion: prompt.Version, // actual version
						attribute.Output:        util.ToJSON(prompt),
					})
				}
				if err != nil {
					promptHubSpan.SetStatusCode(ctx, util.GetErrorCode(err))
					promptHubSpan.SetError(ctx, err.Error())
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
	if cached, ok := p.cache.Get(param.PromptKey, param.Version); ok {
		return cached, nil
	}

	// Cache miss, fetch from server
	promptResults, err := p.openAPIClient.MPullPrompt(ctx, MPullPromptRequest{
		WorkSpaceID: p.config.WorkspaceID,
		Queries: []PromptQuery{
			{
				PromptKey: param.PromptKey,
				Version:   param.Version,
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
	p.cache.Set(promptResults[0].Query.PromptKey, promptResults[0].Query.Version, result)

	return result, nil
}

func (p *Provider) PromptFormat(ctx context.Context, prompt *entity.Prompt, variables map[string]any, options PromptFormatOptions) (messages []*entity.Message, err error) {
	if prompt == nil || prompt.PromptTemplate == nil {
		return nil, nil
	}
	if p.config.PromptTrace && p.traceProvider != nil {
		var promptTemplateSpan *trace.Span
		var spanErr error
		ctx, promptTemplateSpan, spanErr = p.traceProvider.StartSpan(ctx, consts.TracePromptTemplateSpanName, attribute.VPromptTemplateSpanType,
			trace.StartSpanOptions{Scene: attribute.VScenePromptTemplate})
		if spanErr != nil {
			logger.CtxWarnf(ctx, "start prompt template span failed: %v", err)
		}
		defer func() {
			if promptTemplateSpan != nil {
				promptTemplateSpan.SetTags(ctx, map[string]any{
					attribute.PromptKey:     prompt.PromptKey,
					attribute.PromptVersion: prompt.Version,
					attribute.Input:         util.ToJSON(toSpanPromptInput(prompt.PromptTemplate.Messages, variables)),
					attribute.Output:        util.ToJSON(toSpanMessages(messages)),
				})
				if err != nil {
					promptTemplateSpan.SetStatusCode(ctx, util.GetErrorCode(err))
					promptTemplateSpan.SetError(ctx, err.Error())
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
		results = append(results, message)
	}
	return results, nil
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
