// Copyright (c) 2025 Bytedance Ltd. and/or its affiliates
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"bytes"
	"fmt"

	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/exec"
	"github.com/nikolalohinski/gonja/v2/nodes"
	"github.com/nikolalohinski/gonja/v2/parser"

	"github.com/coze-dev/cozeloop-go/internal/consts"
)

func init() {
	// 安全初始化 gonja v2，禁用危险的控制结构
	nilParser := func(p *parser.Parser, args *parser.Parser) (nodes.ControlStructure, error) {
		return nil, fmt.Errorf("invalid statement")
	}
	gonja.DefaultEnvironment.ControlStructures.Replace("include", nilParser)
	gonja.DefaultEnvironment.ControlStructures.Replace("extends", nilParser)
	gonja.DefaultEnvironment.ControlStructures.Replace("import", nilParser)
	gonja.DefaultEnvironment.ControlStructures.Replace("from", nilParser)
}

func InterpolateJinja2(templateStr string, valMap map[string]any) (string, error) {
	// 解析模板
	tpl, err := gonja.FromString(templateStr)
	if err != nil {
		return "", consts.ErrTemplateRender.Wrap(fmt.Errorf("template render error err: %v", err.Error()))
	}

	// 创建执行上下文
	data := exec.NewContext(valMap)
	var out bytes.Buffer

	// 执行模板渲染
	err = tpl.Execute(&out, data)
	if err != nil {
		return "", consts.ErrTemplateRender.Wrap(fmt.Errorf("template render error err: %v", err.Error()))
	}

	return out.String(), nil
}
