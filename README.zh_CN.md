
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

可通过 `TraceQueueConf.ExportConcurrency` 配置每个队列的批次上报并发度：

```go
client, err := loop.NewClient(
    loop.WithTraceQueueConf(&loop.TraceQueueConf{
        ExportConcurrency: 4,
    }),
)
if err != nil {
    panic(err)
}
defer client.Close(context.Background())
```

未配置或值小于等于 `1` 时，保持各队列内串行上报。大于 `1` 时，Span、文件及各自的重试队列分别采用该并发上限，批次完成顺序不保证。队列容量、批次大小和重试次数保持原有配置；队列满时仍会丢弃新数据。`Flush` 和 `Close` 会在 context 允许的时间内等待已发起的上报。自定义 Exporter 和事件回调需要支持并发调用。

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
