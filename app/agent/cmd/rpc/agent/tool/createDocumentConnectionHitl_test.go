package tool

import (
	"context"
	"errors"
	"testing"

	agentcore "github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/agent"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/approval"
	"github.com/Bluore/mutidoc/app/agent/cmd/rpc/agent/botconn"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// hitlModel 首次生成时调用指定工具, 收到工具结果后收尾
type hitlModel struct {
	calls     int
	toolName  string
	arguments string
	answer    string
}

func (m *hitlModel) WithTools(_ []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

func (m *hitlModel) Generate(_ context.Context, input []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	m.calls++
	answer := m.answer
	if answer == "" {
		answer = "编辑连接已建立"
	}
	if n := len(input); n > 0 && input[n-1].Role == schema.Tool {
		return schema.AssistantMessage(answer, nil), nil
	}
	if m.calls == 1 {
		name := m.toolName
		if name == "" {
			name = CreateDocumentConnectionToolName
		}
		args := m.arguments
		if args == "" {
			args = `{"document_id":"d1"}`
		}
		return schema.AssistantMessage("", []schema.ToolCall{{
			ID:       "call-1",
			Type:     "function",
			Function: schema.FunctionCall{Name: name, Arguments: args},
		}}), nil
	}
	return schema.AssistantMessage(answer, nil), nil
}

func (m *hitlModel) Stream(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("stream not supported")
}

// memCheckPointStore 内存检查点存储
type memCheckPointStore struct {
	data map[string][]byte
}

func (s *memCheckPointStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	v, ok := s.data[key]
	return v, ok, nil
}

func (s *memCheckPointStore) Set(_ context.Context, key string, value []byte) error {
	s.data[key] = value
	return nil
}

// TestCreateDocumentConnectionInterruptAndResume 验证工具打断后, 批准恢复才建立连接
func TestCreateDocumentConnectionInterruptAndResume(t *testing.T) {
	ctx := userCtx()
	m := &hitlModel{}
	rpc := &fakeEditRpc{}
	connectionTool, err := NewCreateDocumentConnectionTool(rpc)
	if err != nil {
		t.Fatalf("new tool: %v", err)
	}
	editAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "EditAgent",
		Description: "edit agent",
		Instruction: "edit",
		Model:       m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{connectionTool},
			},
		},
	})
	if err != nil {
		t.Fatalf("new agent: %v", err)
	}

	store := &memCheckPointStore{data: make(map[string][]byte)}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           editAgent,
		EnableStreaming: false,
		CheckPointStore: store,
	})

	iter := runner.Run(ctx, []*schema.Message{schema.UserMessage("编辑文档 d1")}, adk.WithCheckPointID("hitl-cp"))
	interruptID := drainForInterrupt(t, iter)
	if interruptID == "" {
		t.Fatal("agent should be interrupted before creating bot connection")
	}
	if rpc.createCalled {
		t.Fatal("CreateBot should not be called before approval")
	}

	resumeIter, err := runner.ResumeWithParams(ctx, "hitl-cp", &adk.ResumeParams{
		Targets: map[string]any{interruptID: &approval.Decision{Approved: true}},
	})
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	drainEvents(t, resumeIter)
	if !rpc.createCalled {
		t.Fatal("CreateBot should be called after approval")
	}
}

// TestNestedAgentToolInterruptAndResume 验证子智能体工具链路上的打断能恢复
func TestNestedAgentToolInterruptAndResume(t *testing.T) {
	rpc, quitCalled := runNestedHitl(t, true)
	if !rpc.createCalled {
		t.Fatal("CreateBot should be called after approval")
	}
	if !quitCalled {
		t.Fatal("bot connection should be auto-quit after resumed agent finishes")
	}
}

// TestNestedAgentToolReject 验证用户拒绝后不建立连接
func TestNestedAgentToolReject(t *testing.T) {
	rpc, quitCalled := runNestedHitl(t, false)
	if rpc.createCalled {
		t.Fatal("CreateBot should not be called after rejection")
	}
	if quitCalled {
		t.Fatal("no bot connection should be quit after rejection")
	}
}

// runNestedHitl 运行一次嵌套智能体打断并提交审批, 返回 RPC 与自动退出标记
func runNestedHitl(t *testing.T, approved bool) (*fakeEditRpc, bool) {
	t.Helper()

	ctx := botconn.WithTracker(userCtx())
	rpc := &fakeEditRpc{}
	quitCalled := false
	connectionTool, err := NewCreateDocumentConnectionTool(rpc)
	if err != nil {
		t.Fatalf("new tool: %v", err)
	}

	// 子智能体 EditAgent: 调用 createDocumentConnection
	innerAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "EditAgent",
		Description: "edit agent",
		Instruction: "edit",
		Model:       &hitlModel{},
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{connectionTool},
			},
		},
	})
	if err != nil {
		t.Fatalf("new inner agent: %v", err)
	}

	// 外层 MuDo: 通过 agent tool 调用 EditAgent; 与生产一致地包一层自动退出连接
	editAgentTool := agentcore.NewAutoQuitAgentTool(
		adk.NewAgentTool(ctx, innerAgent),
		func(context.Context, string) error {
			quitCalled = true
			return nil
		},
	)

	outerAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "MuDo",
		Description: "main agent",
		Instruction: "main",
		Model: &hitlModel{
			toolName:  "EditAgent",
			arguments: `{"request":"编辑文档 d1"}`,
			answer:    "已完成编辑",
		},
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{editAgentTool},
			},
		},
	})
	if err != nil {
		t.Fatalf("new outer agent: %v", err)
	}

	store := &memCheckPointStore{data: make(map[string][]byte)}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           outerAgent,
		EnableStreaming: false,
		CheckPointStore: store,
	})

	iter := runner.Run(ctx, []*schema.Message{schema.UserMessage("帮我编辑文档 d1")}, adk.WithCheckPointID("hitl-nested-cp"))
	interruptID := drainForInterrupt(t, iter)
	if interruptID == "" {
		t.Fatal("nested agent should be interrupted before creating bot connection")
	}
	if rpc.createCalled {
		t.Fatal("CreateBot should not be called before approval")
	}

	resumeIter, err := runner.ResumeWithParams(ctx, "hitl-nested-cp", &adk.ResumeParams{
		Targets: map[string]any{interruptID: &approval.Decision{Approved: approved}},
	})
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	drainEvents(t, resumeIter)

	return rpc, quitCalled
}

// drainForInterrupt 消费事件直到打断发生, 返回根因打断点 ID
func drainForInterrupt(t *testing.T, iter *adk.AsyncIterator[*adk.AgentEvent]) string {
	t.Helper()
	for {
		event, ok := iter.Next()
		if !ok {
			return ""
		}
		if event.Err != nil {
			t.Fatalf("event error: %v", event.Err)
		}
		if event.Action == nil || event.Action.Interrupted == nil {
			continue
		}
		for _, ic := range event.Action.Interrupted.InterruptContexts {
			if ic != nil && ic.IsRootCause {
				return ic.ID
			}
		}
	}
}

// drainEvents 消费剩余事件, 遇到错误直接失败
func drainEvents(t *testing.T, iter *adk.AsyncIterator[*adk.AgentEvent]) {
	t.Helper()
	for {
		event, ok := iter.Next()
		if !ok {
			return
		}
		if event.Err != nil {
			t.Fatalf("event error: %v", event.Err)
		}
	}
}
