# CozeLoop Go SDK
[English](README.md) | [简体中文](README.zh_CN.md)

## Overview

The CozeLoop SDK is a Go client for interacting with [CozeLoop platform](https://loop.coze.cn).
Key features:
- Report trace
- Get and format prompt

## Requirement
- Go 1.18 or higher

## Installation

`go get github.com/coze-dev/cozeloop-go`

## Usage

### Initialize

To get started, visit https://loop.coze.cn/console/enterprise/personal/open/oauth/apps and create an OAuth app.
Then you can get your owner appid, public key and private key.

Set your environment variables:
```bash
export COZELOOP_WORKSPACE_ID=your workspace id
export COZELOOP_JWT_OAUTH_CLIENT_ID=your client id
export COZELOOP_JWT_OAUTH_PRIVATE_KEY=your private key
export COZELOOP_JWT_OAUTH_PUBLIC_KEY_ID=your public key id
```

### Report Trace

```go
func main() {
    ctx, span := loop.StartSpan(ctx, "root", "custom")

    span.SetInput(ctx, "Hello") 
    span.SetOutput(ctx, "World") 
	
    span.Finish(ctx)
	
    loop.Close(ctx)
}
```

Set `TraceQueueConf.ExportConcurrency` to configure concurrent batch exports per queue:

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

Unset values and values <= `1` preserve serial exports within each queue. Higher values apply independently to the span, file, and respective retry queues; batches may complete out of order. Queue capacity, batch size, and retry counts retain their existing settings, and full queues still drop new items. `Flush` and `Close` wait for in-flight exports within the context deadline. Custom exporters and event callbacks must support concurrent calls.

### Get Prompt
```go
func main() {
    prompt, err := loop.GetPrompt(ctx, loop.GetPromptParam{PromptKey: "your_prompt_key"})
    messages, err := loop.PromptFormat(ctx, prompt, map[string]any{
        "var1": "your content",
    })
}
```

You can see more examples [here](examples).

## Contribution

Please check [Contributing](CONTRIBUTING.md) for more details.

## Security

If you discover a potential security issue in this project, or think you may
have discovered a security issue, we ask that you notify Bytedance Security via our [security center](https://security.bytedance.com/src) or [vulnerability reporting email](sec@bytedance.com).

Please do **not** create a public GitHub issue.

## License

This project is licensed under the [MIT License](LICENSE).
