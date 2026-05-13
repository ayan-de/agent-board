package orchestrator

import (
	"github.com/ayan-de/agent-board/internal/core"
)

type CreateProposalInput = core.CreateProposalInput
type ApplyRunOutcomeInput = core.ApplyRunOutcomeInput
type FinishRunInput = core.FinishRunInput
type RunRequest = core.RunRequest
type RunHandle = core.RunHandle
type RunCompletion = core.RunCompletion
type AgentSession = core.AgentSession

type LLMClient = core.LLMClient
type Runner = core.Runner
type AgentRunner = core.AgentRunner
type Store = core.Store
type ContextCarryProvider = core.ContextCarryProvider
