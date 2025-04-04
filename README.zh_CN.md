
# CozeLoop Go SDK
[English](README.md) | 简体中文

## 概述

CozeLoop SDK 是一个用于与 [扣子罗盘平台](https://loop.coze.cn) 进行交互的 Go 客户端。
主要功能：
- Trace上报
- Prompt拉取

## 要求
- Go 1.18 或更高版本

## 安装

`go get github.com/coze-dev/cozeloop-go`

## 用法

### 初始化

首先，访问 https://loop.coze.cn/console/enterprise/personal/open/oauth/apps 并创建一个 OAuth 应用，
获取应用所有者的 AppID、公钥和私钥。

设置环境变量
```bash
export COZELOOP_WORKSPACE_ID=your workspace id
export COZELOOP_JWT_OAUTH_CLIENT_ID=your client id
export COZELOOP_JWT_OAUTH_PRIVATE_KEY=your private key
export COZELOOP_JWT_OAUTH_PUBLIC_KEY_ID=your public key id
```

### Trace上报

```go
func main() {
    ctx, span := loop.StartSpan(ctx, "root", "custom")

    span.SetInput(ctx, "Hello") 
    span.SetOutput(ctx, "World") 
	
    span.Finish(ctx)
	
    loop.Close(ctx)
}
```

### Prompt拉取
```go
func main() {
    prompt, err := loop.GetPrompt(ctx, loop.GetPromptParam{PromptKey: "your_prompt_key"})
    messages, err := loop.PromptFormat(ctx, prompt, map[string]any{
        "var1": "your content",
    })
}
```

你可以在 [这里](examples) 查看更多示例。


## 贡献

如需了解更多详细信息，请查看 [Contributing](CONTRIBUTING.md)。


## 安全

如果你发现本项目中存在潜在的安全问题，或者认为自己可能发现了安全问题，请通过我们的 [安全中心](https://security.bytedance.com/src) 或 [漏洞报告邮箱](sec@bytedance.com) 通知字节跳动安全团队。
请**不要**创建公开的 GitHub 问题。

## License

本项目采用 [MIT License](LICENSE)。