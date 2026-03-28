package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ai-gateway/pi-go/agent"
	"github.com/ai-gateway/pi-go/auth"
	"github.com/ai-gateway/pi-go/provider/anthropic"
	"github.com/ai-gateway/pi-go/tool"
	"github.com/ai-gateway/pi-go/types"
)

func main() {
	switch os.Args[1] {
	case "1":
		example1BasicChat()
	case "2":
		example2WithTool()
	case "3":
		example3Events()
	}
}

// ─────────────────────────────────────────────
// 例子 1：最简单的单轮对话
// ─────────────────────────────────────────────
func example1BasicChat() {
	// 1. 认证：从环境变量读取 API Key
	storage := auth.NewAuthStorage("")
	resolver := auth.NewAuthResolver(storage)

	apiKey, err := resolver.ResolveAPIKey(context.Background(), "anthropic")
	if err != nil || apiKey == "" {
		fmt.Println("请设置 ANTHROPIC_API_KEY 环境变量")
		os.Exit(1)
	}

	// 2. 创建 Provider
	models := anthropic.BuiltinModels()
	prov := anthropic.New(apiKey)

	// 3. 创建 Agent
	a := agent.NewAgent(prov,
		agent.WithModel(models[1]), // claude-sonnet-4-6
		agent.WithSystemPrompt("你是一个有帮助的助手，用中文回复。"),
	)

	// 4. 订阅事件，流式打印输出
	a.Subscribe(func(ev types.AgentEvent) {
		if ev.Type == types.EventMessageUpdate && ev.StreamEvent != nil {
			if ev.StreamEvent.Type == types.StreamEventTextDelta {
				fmt.Print(ev.StreamEvent.Delta)
			}
		}
		if ev.Type == types.EventAgentEnd {
			fmt.Println()
		}
	})

	// 5. 发送消息，等待完成
	ctx := context.Background()
	if err := a.Prompt(ctx, "用一句话介绍 Go 语言"); err != nil {
		fmt.Println("错误:", err)
		os.Exit(1)
	}
	a.WaitForIdle(ctx)
}

// ─────────────────────────────────────────────
// 例子 2：带 Tool 的多轮对话
// ─────────────────────────────────────────────
func example2WithTool() {
	storage := auth.NewAuthStorage("")
	resolver := auth.NewAuthResolver(storage)
	apiKey, _ := resolver.ResolveAPIKey(context.Background(), "anthropic")

	prov := anthropic.New(apiKey)
	models := anthropic.BuiltinModels()

	// 定义一个"查天气"工具
	weatherTool := &tool.BaseToolImpl{
		ToolName:        "get_weather",
		ToolDescription: "查询指定城市的当前天气",
		ToolLabel:       "查天气",
		ToolSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"city": map[string]any{
					"type":        "string",
					"description": "城市名称，如 Beijing",
				},
			},
			"required": []string{"city"},
		},
		ExecuteFn: func(ctx context.Context, id string, params map[string]any, onUpdate func(tool.ToolUpdate)) (tool.ToolResult, error) {
			city := params["city"].(string)
			result := fmt.Sprintf("%s 今天晴，气温 22°C，湿度 60%%", city)
			return tool.ToolResult{
				Content: []types.ContentBlock{
					&types.TextContent{Type: types.ContentTypeText, Text: result},
				},
			}, nil
		},
	}

	a := agent.NewAgent(prov,
		agent.WithModel(models[1]),
		agent.WithSystemPrompt("你是天气助手。需要查天气时使用 get_weather 工具。"),
		agent.WithTools([]tool.AgentTool{weatherTool}),
	)

	// 打印 AI 输出 + tool 调用情况
	a.Subscribe(func(ev types.AgentEvent) {
		switch ev.Type {
		case types.EventToolExecutionStart:
			fmt.Printf("\n[调用工具: %s, 参数: %v]\n", ev.ToolName, ev.ToolArgs)
		case types.EventMessageUpdate:
			if ev.StreamEvent != nil && ev.StreamEvent.Type == types.StreamEventTextDelta {
				fmt.Print(ev.StreamEvent.Delta)
			}
		case types.EventAgentEnd:
			fmt.Println()
		}
	})

	ctx := context.Background()
	a.Prompt(ctx, "北京今天天气怎么样？")
	a.WaitForIdle(ctx)
}

// ─────────────────────────────────────────────
// 例子 3：Steering（中途注入消息）
// ─────────────────────────────────────────────
func example3Events() {
	storage := auth.NewAuthStorage("")
	resolver := auth.NewAuthResolver(storage)
	apiKey, _ := resolver.ResolveAPIKey(context.Background(), "anthropic")

	prov := anthropic.New(apiKey)
	models := anthropic.BuiltinModels()

	a := agent.NewAgent(prov,
		agent.WithModel(models[2]), // claude-haiku（便宜）
		agent.WithSystemPrompt("你是一个助手。"),
	)

	// 收集所有事件类型
	a.Subscribe(func(ev types.AgentEvent) {
		switch ev.Type {
		case types.EventAgentStart:
			fmt.Println(">>> Agent 开始")
		case types.EventTurnStart:
			fmt.Println("  > 新一轮 LLM 调用")
		case types.EventMessageUpdate:
			if ev.StreamEvent != nil && ev.StreamEvent.Type == types.StreamEventTextDelta {
				fmt.Print(ev.StreamEvent.Delta)
			}
		case types.EventTurnEnd:
			fmt.Println()
		case types.EventAgentEnd:
			fmt.Println(">>> Agent 结束")
		}
	})

	ctx := context.Background()

	// 第一个问题
	a.Prompt(ctx, "1+1 等于几？")

	// 在 agent 运行中注入 follow-up（等第一轮结束后自动继续）
	a.FollowUp(&types.UserMessage{
		Role: "user",
		Content: []types.ContentBlock{
			&types.TextContent{Type: types.ContentTypeText, Text: "那 2+2 呢？"},
		},
	})

	a.WaitForIdle(ctx)
}
