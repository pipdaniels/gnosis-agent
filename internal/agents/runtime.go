package agents

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AgentRuntime manages all agents in the system
type AgentRuntime struct {
	agents map[string]Agent
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// Agent interface that all agents must implement
type Agent interface {
	Name() string
	Initialize(ctx context.Context, config interface{}) error
	Execute(ctx context.Context, input interface{}) (interface{}, error)
	Shutdown(ctx context.Context) error
}

// ExecutionResult wraps agent execution results
type ExecutionResult struct {
	AgentName     string
	Input         interface{}
	Output        interface{}
	Error         error
	ExecutionTime time.Duration
	Reasoning     []string
}

// NewAgentRuntime creates a new agent runtime
func NewAgentRuntime(ctx context.Context) *AgentRuntime {
	runtimeCtx, cancel := context.WithCancel(ctx)
	return &AgentRuntime{
		agents: make(map[string]Agent),
		ctx:    runtimeCtx,
		cancel: cancel,
	}
}

// Register registers an agent with the runtime
func (r *AgentRuntime) Register(agent Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := agent.Name()
	if _, exists := r.agents[name]; exists {
		return fmt.Errorf("agent %s already registered", name)
	}

	r.agents[name] = agent
	return nil
}

// Get retrieves an agent by name
func (r *AgentRuntime) Get(name string) (Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, exists := r.agents[name]
	if !exists {
		return nil, fmt.Errorf("agent %s not found", name)
	}

	return agent, nil
}

// Execute runs a specific agent
func (r *AgentRuntime) Execute(ctx context.Context, agentName string, input interface{}) (*ExecutionResult, error) {
	r.mu.RLock()
	agent, exists := r.agents[agentName]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("agent %s not found", agentName)
	}

	start := time.Now()
	output, err := agent.Execute(ctx, input)
	duration := time.Since(start)

	return &ExecutionResult{
		AgentName:     agentName,
		Input:         input,
		Output:        output,
		Error:         err,
		ExecutionTime: duration,
	}, nil
}

// ExecutePipeline runs multiple agents in sequence
// Each agent's output becomes the next agent's input
func (r *AgentRuntime) ExecutePipeline(ctx context.Context, agentNames []string, input interface{}) ([]*ExecutionResult, error) {
	results := make([]*ExecutionResult, 0, len(agentNames))
	currentInput := input

	for _, agentName := range agentNames {
		result, err := r.Execute(ctx, agentName, currentInput)
		if err != nil {
			return results, fmt.Errorf("pipeline failed at %s: %w", agentName, err)
		}

		results = append(results, result)

		// Check if agent execution had an error
		if result.Error != nil {
			return results, fmt.Errorf("agent %s failed: %w", agentName, result.Error)
		}

		currentInput = result.Output
	}

	return results, nil
}

// ExecuteParallel runs multiple agents in parallel with the same input
func (r *AgentRuntime) ExecuteParallel(ctx context.Context, agentNames []string, input interface{}) ([]*ExecutionResult, error) {
	results := make([]*ExecutionResult, len(agentNames))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i, agentName := range agentNames {
		wg.Add(1)
		go func(idx int, name string) {
			defer wg.Done()

			result, err := r.Execute(ctx, name, input)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}

			results[idx] = result
		}(i, agentName)
	}

	wg.Wait()

	if firstErr != nil {
		return results, firstErr
	}

	return results, nil
}

// ListAgents returns the names of all registered agents
func (r *AgentRuntime) ListAgents() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.agents))
	for name := range r.agents {
		names = append(names, name)
	}

	return names
}

// Shutdown shuts down all agents
func (r *AgentRuntime) Shutdown() error {
	r.cancel()

	r.mu.RLock()
	defer r.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var errors []error
	for _, agent := range r.agents {
		if err := agent.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", agent.Name(), err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	return nil
}
