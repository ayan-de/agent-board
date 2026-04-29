# Graph Report - .  (2026-04-29)

## Corpus Check
- 146 files · ~113,619 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 973 nodes · 2186 edges · 69 communities detected
- Extraction: 53% EXTRACTED · 47% INFERRED · 0% AMBIGUOUS · INFERRED: 1017 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Test Cases (Actions & Approvals)|Test Cases (Actions & Approvals)]]
- [[_COMMUNITY_Agent Exec Runner|Agent Exec Runner]]
- [[_COMMUNITY_Agent Detection|Agent Detection]]
- [[_COMMUNITY_App Test Infrastructure|App Test Infrastructure]]
- [[_COMMUNITY_Config Structs|Config Structs]]
- [[_COMMUNITY_ClaudeCodex Agent Formatting|Claude/Codex Agent Formatting]]
- [[_COMMUNITY_MCP Client Manager|MCP Client Manager]]
- [[_COMMUNITY_App Commands & Initialization|App Commands & Initialization]]
- [[_COMMUNITY_TUI Components (Modal, Palette, Theme)|TUI Components (Modal, Palette, Theme)]]
- [[_COMMUNITY_Kanban Board|Kanban Board]]
- [[_COMMUNITY_Ticket View|Ticket View]]
- [[_COMMUNITY_Notification System|Notification System]]
- [[_COMMUNITY_Dashboard & Agent Utils|Dashboard & Agent Utils]]
- [[_COMMUNITY_LLM Provider Factory|LLM Provider Factory]]
- [[_COMMUNITY_Orchestrator Core (Proposal, Pane, Approval)|Orchestrator Core (Proposal, Pane, Approval)]]
- [[_COMMUNITY_App Rendering (ANSI, Overlay)|App Rendering (ANSI, Overlay)]]
- [[_COMMUNITY_Orchestrator Test Infrastructure|Orchestrator Test Infrastructure]]
- [[_COMMUNITY_LangChain LLM Integration|LangChain LLM Integration]]
- [[_COMMUNITY_Text Input Modal|Text Input Modal]]
- [[_COMMUNITY_Orchestrator Types|Orchestrator Types]]
- [[_COMMUNITY_Scratch Config Utils|Scratch Config Utils]]
- [[_COMMUNITY_Documentation (AGENTS & Roadmap)|Documentation (AGENTS & Roadmap)]]
- [[_COMMUNITY_Keybinding Module|Keybinding Module]]
- [[_COMMUNITY_Pane Module|Pane Module]]
- [[_COMMUNITY_Approval Action|Approval Action]]
- [[_COMMUNITY_Orchestrator Actions|Orchestrator Actions]]
- [[_COMMUNITY_Summarizer|Summarizer]]
- [[_COMMUNITY_MCP Session Carry|MCP Session Carry]]
- [[_COMMUNITY_API Server|API Server]]
- [[_COMMUNITY_API Handlers|API Handlers]]
- [[_COMMUNITY_API WebSocket|API WebSocket]]
- [[_COMMUNITY_API Middleware|API Middleware]]
- [[_COMMUNITY_Store Migrations|Store Migrations]]
- [[_COMMUNITY_Decomposer|Decomposer]]
- [[_COMMUNITY_Assigner|Assigner]]
- [[_COMMUNITY_Decomposition Prompts|Decomposition Prompts]]
- [[_COMMUNITY_API Ticket Types|API Ticket Types]]
- [[_COMMUNITY_API Session Types|API Session Types]]
- [[_COMMUNITY_API Agent Types|API Agent Types]]
- [[_COMMUNITY_Set Pty Runner|Set Pty Runner]]
- [[_COMMUNITY_Get Active Sessions|Get Active Sessions]]
- [[_COMMUNITY_Get Active Session By Ticket|Get Active Session By Ticket]]
- [[_COMMUNITY_Get Active Session By Agent|Get Active Session By Agent]]
- [[_COMMUNITY_Append Log|Append Log]]
- [[_COMMUNITY_Get Logs|Get Logs]]
- [[_COMMUNITY_Get Tmux Runner|Get Tmux Runner]]
- [[_COMMUNITY_New Exec Runner|New Exec Runner]]
- [[_COMMUNITY_Run Handle|Run Handle]]
- [[_COMMUNITY_Run Completion|Run Completion]]
- [[_COMMUNITY_Create Proposal Input|Create Proposal Input]]
- [[_COMMUNITY_Apply Run Outcome Input|Apply Run Outcome Input]]
- [[_COMMUNITY_Finish Run Input|Finish Run Input]]
- [[_COMMUNITY_Get Pane|Get Pane]]
- [[_COMMUNITY_List Panes|List Panes]]
- [[_COMMUNITY_List Panes By Agent|List Panes By Agent]]
- [[_COMMUNITY_Send Input (Pane)|Send Input (Pane)]]
- [[_COMMUNITY_Switch To Pane|Switch To Pane]]
- [[_COMMUNITY_Capture Pane|Capture Pane]]
- [[_COMMUNITY_Start Tmux|Start Tmux]]
- [[_COMMUNITY_Get Pane Manager|Get Pane Manager]]
- [[_COMMUNITY_Send Input (Tmux)|Send Input (Tmux)]]
- [[_COMMUNITY_Capture Pane (Tmux)|Capture Pane (Tmux)]]
- [[_COMMUNITY_List Panes (Tmux)|List Panes (Tmux)]]
- [[_COMMUNITY_Get Runner|Get Runner]]
- [[_COMMUNITY_Get Pane ID|Get Pane ID]]
- [[_COMMUNITY_Run Agent|Run Agent]]
- [[_COMMUNITY_Coordinator Test ContextCarry|Coordinator Test: ContextCarry]]
- [[_COMMUNITY_Coordinator Test Events|Coordinator Test: Events]]
- [[_COMMUNITY_Coordinator Test Unassigned Ticket|Coordinator Test: Unassigned Ticket]]

## God Nodes (most connected - your core abstractions)
1. `openTestDB()` - 46 edges
2. `Open()` - 36 edges
3. `newTestTicketView()` - 35 edges
4. `newTestApp()` - 32 edges
5. `Store` - 32 edges
6. `newTestKanban()` - 30 edges
7. `DefaultKeyMap()` - 25 edges
8. `newTestDashboard()` - 24 edges
9. `Command` - 22 edges
10. `main()` - 21 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `NewManager()`  [INFERRED]
  cmd/agentboard/main.go → internal/mcp/client.go
- `main()` --calls--> `Load()`  [INFERRED]
  cmd/agentboard/main.go → internal/config/config.go
- `main()` --calls--> `Open()`  [INFERRED]
  cmd/agentboard/main.go → internal/store/sqlite.go
- `main()` --calls--> `NewFromConfig()`  [INFERRED]
  cmd/agentboard/main.go → internal/llm/factory.go
- `main()` --calls--> `NewExecRunner()`  [INFERRED]
  cmd/agentboard/main.go → internal/orchestrator/exec_runner.go

## Hyperedges (group relationships)
- **store persistence domain objects** — store_ticket, store_session, store_proposal, store_event, store_contextcarry [EXTRACTED 1.00]
- **config sub-configuration structs** — config_generalconfig, config_boardconfig, config_agentconfig, config_tuiconfig, config_llmconfig, config_dbconfig, config_mcpconfig [EXTRACTED 1.00]
- **ticket-related data structures** — store_ticket, store_ticketfilters, store_ticketrow, store_session, store_proposal, store_event, store_contextcarry [INFERRED 0.80]
- **Runner interface implementations** — types_runner, exec_runner_execrunner, tmux_runner_tmuxrunner, pty_runner_tyrunner [EXTRACTED 1.00]
- **Proposal lifecycle pipeline** — service_createproposal, approval_approveproposal, service_startapprovedrun, summarizer_finishrun [EXTRACTED 0.90]
- **Pane monitoring implementations** — pane_manager_monitorpane, pty_runner_monitorpane, pane_manager_createpane, pty_runner_injectprompt [EXTRACTED 0.85]

## Communities

### Community 0 - "Test Cases (Actions & Approvals)"
Cohesion: 0.05
Nodes (69): TestApplyRunOutcomeMovesTicketToReview(), TestApproveProposalRejectsNonPending(), TestApproveProposalRejectsStaleTicketState(), TestApproveProposalSucceeds(), TestCreateProposalRecordsEvent(), TestCreateProposalRejectsUnassignedTicket(), TestCreateProposalUsesTicketAndContextCarry(), AgentSession (+61 more)

### Community 1 - "Agent Exec Runner"
Cohesion: 0.03
Nodes (42): TestActionString(), formatUptime(), NewExecRunner(), opencodeEvent, ParseOpencodeOutput(), TestExecRunnerAgentNotFound(), TestParseOpencodeOutputCompleted(), TestParseOpencodeOutputEmpty() (+34 more)

### Community 2 - "Agent Detection"
Cohesion: 0.06
Nodes (47): AgentColor(), DetectAgents(), TestAgentColor(), TestDetectAgents(), TestDetectAgentsFoundWithBinaryOnPath(), TestDetectAgentsNotFoundWhenMissing(), agentSpec, DetectedAgent (+39 more)

### Community 3 - "App Test Infrastructure"
Cohesion: 0.08
Nodes (49): execCmd(), newTestApp(), TestAppDashboardViewRenders(), TestAppEscapeFromDashboard(), TestAppEscapeReturnsToBoard(), TestAppForceQuit(), TestAppNavigationDelegatesToKanban(), TestAppOpenTicket() (+41 more)

### Community 4 - "Config Structs"
Cohesion: 0.05
Nodes (43): DefaultPrefix(), AgentConfig, BoardConfig, Config, DBConfig, GeneralConfig, getGitRemote(), getGitRemote (+35 more)

### Community 5 - "Claude/Codex Agent Formatting"
Cohesion: 0.07
Nodes (36): NewClaudeCode(), NewCodex(), NewGeminiCode(), NewOpenCode(), NewRegistry(), registerBuiltins(), NewCommandRegistry(), TestCommandRegistry() (+28 more)

### Community 6 - "MCP Client Manager"
Cohesion: 0.06
Nodes (27): NewClient(), NewManager(), MCPConfig, MCPServerConfig, NewContextCarryAdapter(), TestContextCarryAdapterBuildsHandoffPayload(), Client, ProposalDraft (+19 more)

### Community 7 - "App Commands & Initialization"
Cohesion: 0.1
Nodes (28): newAppCommands(), NewApp(), ApplyConfig(), TestActionNamesMap(), TestApplyConfigEmptyMap(), TestApplyConfigGoToTicketPrefix(), TestApplyConfigReplacesAllBindingsForAction(), TestApplyConfigSingleOverride() (+20 more)

### Community 8 - "TUI Components (Modal, Palette, Theme)"
Cohesion: 0.06
Nodes (14): GetBaseDir(), SaveTheme(), NewConfirmModalStyles(), NewCommandPalette(), NewTicketViewModel(), NewTicketViewStyles(), parseTags(), ticketFields() (+6 more)

### Community 9 - "Kanban Board"
Cohesion: 0.12
Nodes (30): DefaultKanbanStyles(), NewKanbanModel(), NewKanbanStyles(), newTestKanban(), TestDefaultKanbanStyles(), TestKanbanAddTicket(), TestKanbanColumnNavigation(), TestKanbanDeleteTicket() (+22 more)

### Community 10 - "Ticket View"
Cohesion: 0.17
Nodes (29): newTestTicketView(), TestNewTicketViewModel(), TestTicketViewModelAgentSelectAssign(), TestTicketViewModelAgentSelectCancel(), TestTicketViewModelAgentSelectDropdownRenders(), TestTicketViewModelAgentSelectMode(), TestTicketViewModelAgentSelectNavigate(), TestTicketViewModelAgentSelectNone() (+21 more)

### Community 11 - "Notification System"
Cohesion: 0.11
Nodes (18): NewNotificationStyles(), newTestNotification(), TestNotificationDismissIgnoresStaleMessage(), TestNotificationDismissMessageRemovesMatchingNotification(), TestNotificationDoesNotHandleKeyboardInput(), TestNotificationInactiveByDefault(), TestNotificationShowAppendsItemAndReturnsDismissCmd(), TestNotificationShowKeepsNewestFourItems() (+10 more)

### Community 12 - "Dashboard & Agent Utils"
Cohesion: 0.14
Nodes (25): StripANSI(), newFakeOrchestrator(), NewDashboardModel(), NewDashboardStyles(), newTestDashboard(), parseDuration(), TestDashboardInit(), TestDashboardRefresh() (+17 more)

### Community 13 - "LLM Provider Factory"
Cohesion: 0.09
Nodes (23): GetProvider(), NewFromConfig(), newProviderModel(), Providers(), RegisterProvider(), TestFactoryReturnsClientWithClaude(), TestFactoryReturnsClientWithOllama(), TestFactoryReturnsClientWithOpenAI() (+15 more)

### Community 14 - "Orchestrator Core (Proposal, Pane, Approval)"
Cohesion: 0.1
Nodes (28): ApproveProposal, ExecRunner, Start (ExecRunner), AgentPane, CreatePane, PaneManager, RemovePane, injectPrompt (+20 more)

### Community 15 - "App Rendering (ANSI, Overlay)"
Cohesion: 0.08
Nodes (26): ansiSkip(), ansiTruncate(), overlayLine(), adhocRunStartedMsg, AppDeps, deleteTicketConfirmMsg, deleteTicketRequestMsg, editorFinishedMsg (+18 more)

### Community 16 - "Orchestrator Test Infrastructure"
Cohesion: 0.08
Nodes (5): fakeAsyncRunner, fakeCtx, fakeLLMClient, fakeRunner, fakeStore

### Community 17 - "LangChain LLM Integration"
Cohesion: 0.12
Nodes (11): sanitizeProposal(), TestGenerateProposalBuildsPromptFromTicketContext(), TestGenerateProposalStripsThinkBlocks(), TestSummarizeContextReturnsSummary(), LangChainClient, llmModelFunc, stubModel, GenerateProposal() (+3 more)

### Community 18 - "Text Input Modal"
Cohesion: 0.18
Nodes (3): NewTextInputModalStyles(), TextInputModal, TextInputModalStyles

### Community 19 - "Orchestrator Types"
Cohesion: 0.18
Nodes (10): ApplyRunOutcomeInput, ContextCarryProvider, CreateProposalInput, FinishRunInput, LLMClient, RunCompletion, RunHandle, Runner (+2 more)

### Community 20 - "Scratch Config Utils"
Cohesion: 0.33
Nodes (3): Config, MCPConfig, MCPServerConfig

### Community 21 - "Documentation (AGENTS & Roadmap)"
Cohesion: 1.0
Nodes (1): AgentBoard Roadmap

### Community 22 - "Keybinding Module"
Cohesion: 1.0
Nodes (0): 

### Community 23 - "Pane Module"
Cohesion: 1.0
Nodes (0): 

### Community 24 - "Approval Action"
Cohesion: 1.0
Nodes (0): 

### Community 25 - "Orchestrator Actions"
Cohesion: 1.0
Nodes (0): 

### Community 26 - "Summarizer"
Cohesion: 1.0
Nodes (0): 

### Community 27 - "MCP Session Carry"
Cohesion: 1.0
Nodes (0): 

### Community 28 - "API Server"
Cohesion: 1.0
Nodes (0): 

### Community 29 - "API Handlers"
Cohesion: 1.0
Nodes (0): 

### Community 30 - "API WebSocket"
Cohesion: 1.0
Nodes (0): 

### Community 31 - "API Middleware"
Cohesion: 1.0
Nodes (0): 

### Community 32 - "Store Migrations"
Cohesion: 1.0
Nodes (0): 

### Community 33 - "Decomposer"
Cohesion: 1.0
Nodes (0): 

### Community 34 - "Assigner"
Cohesion: 1.0
Nodes (0): 

### Community 35 - "Decomposition Prompts"
Cohesion: 1.0
Nodes (0): 

### Community 36 - "API Ticket Types"
Cohesion: 1.0
Nodes (0): 

### Community 37 - "API Session Types"
Cohesion: 1.0
Nodes (0): 

### Community 38 - "API Agent Types"
Cohesion: 1.0
Nodes (0): 

### Community 39 - "Set Pty Runner"
Cohesion: 1.0
Nodes (1): SetPtyRunner

### Community 40 - "Get Active Sessions"
Cohesion: 1.0
Nodes (1): GetActiveSessions

### Community 41 - "Get Active Session By Ticket"
Cohesion: 1.0
Nodes (1): GetActiveSessionByTicket

### Community 42 - "Get Active Session By Agent"
Cohesion: 1.0
Nodes (1): GetActiveSessionByAgent

### Community 43 - "Append Log"
Cohesion: 1.0
Nodes (1): AppendLog

### Community 44 - "Get Logs"
Cohesion: 1.0
Nodes (1): GetLogs

### Community 45 - "Get Tmux Runner"
Cohesion: 1.0
Nodes (1): GetTmuxRunner

### Community 46 - "New Exec Runner"
Cohesion: 1.0
Nodes (1): NewExecRunner

### Community 47 - "Run Handle"
Cohesion: 1.0
Nodes (1): RunHandle

### Community 48 - "Run Completion"
Cohesion: 1.0
Nodes (1): RunCompletion

### Community 49 - "Create Proposal Input"
Cohesion: 1.0
Nodes (1): CreateProposalInput

### Community 50 - "Apply Run Outcome Input"
Cohesion: 1.0
Nodes (1): ApplyRunOutcomeInput

### Community 51 - "Finish Run Input"
Cohesion: 1.0
Nodes (1): FinishRunInput

### Community 52 - "Get Pane"
Cohesion: 1.0
Nodes (1): GetPane

### Community 53 - "List Panes"
Cohesion: 1.0
Nodes (1): ListPanes

### Community 54 - "List Panes By Agent"
Cohesion: 1.0
Nodes (1): ListPanesByAgent

### Community 55 - "Send Input (Pane)"
Cohesion: 1.0
Nodes (1): SendInput

### Community 56 - "Switch To Pane"
Cohesion: 1.0
Nodes (1): SwitchToPane

### Community 57 - "Capture Pane"
Cohesion: 1.0
Nodes (1): CapturePane

### Community 58 - "Start Tmux"
Cohesion: 1.0
Nodes (1): Start (TmuxRunner)

### Community 59 - "Get Pane Manager"
Cohesion: 1.0
Nodes (1): GetPaneManager

### Community 60 - "Send Input (Tmux)"
Cohesion: 1.0
Nodes (1): SendInput

### Community 61 - "Capture Pane (Tmux)"
Cohesion: 1.0
Nodes (1): CapturePane

### Community 62 - "List Panes (Tmux)"
Cohesion: 1.0
Nodes (1): ListPanes

### Community 63 - "Get Runner"
Cohesion: 1.0
Nodes (1): GetRunner

### Community 64 - "Get Pane ID"
Cohesion: 1.0
Nodes (1): GetPaneID

### Community 65 - "Run Agent"
Cohesion: 1.0
Nodes (1): RunAgent

### Community 66 - "Coordinator Test: ContextCarry"
Cohesion: 1.0
Nodes (1): TestCreateProposalUsesTicketAndContextCarry

### Community 67 - "Coordinator Test: Events"
Cohesion: 1.0
Nodes (1): TestCreateProposalRecordsEvent

### Community 68 - "Coordinator Test: Unassigned Ticket"
Cohesion: 1.0
Nodes (1): TestCreateProposalRejectsUnassignedTicket

## Ambiguous Edges - Review These
- `GetProjectInitDate()` → `Theme`  [AMBIGUOUS]
  internal/config/config.go · relation: conceptually_related_to

## Knowledge Gaps
- **112 isolated node(s):** `TicketCardStyles`, `tickMsg`, `NotificationVariant`, `notificationDismissMsg`, `ticketCreatedMsg` (+107 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Documentation (AGENTS & Roadmap)`** (2 nodes): `AGENTS.md`, `AgentBoard Roadmap`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Keybinding Module`** (1 nodes): `keybindings.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Pane Module`** (1 nodes): `pane.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Approval Action`** (1 nodes): `approval.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Orchestrator Actions`** (1 nodes): `actions.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Summarizer`** (1 nodes): `summarizer.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `MCP Session Carry`** (1 nodes): `sessioncarry.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `API Server`** (1 nodes): `server.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `API Handlers`** (1 nodes): `handlers.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `API WebSocket`** (1 nodes): `websocket.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `API Middleware`** (1 nodes): `middleware.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Store Migrations`** (1 nodes): `migrations.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Decomposer`** (1 nodes): `decomposer.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Assigner`** (1 nodes): `assigner.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Decomposition Prompts`** (1 nodes): `prompts.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `API Ticket Types`** (1 nodes): `ticket.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `API Session Types`** (1 nodes): `session.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `API Agent Types`** (1 nodes): `agent.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Set Pty Runner`** (1 nodes): `SetPtyRunner`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Get Active Sessions`** (1 nodes): `GetActiveSessions`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Get Active Session By Ticket`** (1 nodes): `GetActiveSessionByTicket`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Get Active Session By Agent`** (1 nodes): `GetActiveSessionByAgent`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Append Log`** (1 nodes): `AppendLog`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Get Logs`** (1 nodes): `GetLogs`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Get Tmux Runner`** (1 nodes): `GetTmuxRunner`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `New Exec Runner`** (1 nodes): `NewExecRunner`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Run Handle`** (1 nodes): `RunHandle`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Run Completion`** (1 nodes): `RunCompletion`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Create Proposal Input`** (1 nodes): `CreateProposalInput`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Apply Run Outcome Input`** (1 nodes): `ApplyRunOutcomeInput`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Finish Run Input`** (1 nodes): `FinishRunInput`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Get Pane`** (1 nodes): `GetPane`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `List Panes`** (1 nodes): `ListPanes`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `List Panes By Agent`** (1 nodes): `ListPanesByAgent`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Send Input (Pane)`** (1 nodes): `SendInput`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Switch To Pane`** (1 nodes): `SwitchToPane`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Capture Pane`** (1 nodes): `CapturePane`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Start Tmux`** (1 nodes): `Start (TmuxRunner)`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Get Pane Manager`** (1 nodes): `GetPaneManager`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Send Input (Tmux)`** (1 nodes): `SendInput`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Capture Pane (Tmux)`** (1 nodes): `CapturePane`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `List Panes (Tmux)`** (1 nodes): `ListPanes`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Get Runner`** (1 nodes): `GetRunner`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Get Pane ID`** (1 nodes): `GetPaneID`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Run Agent`** (1 nodes): `RunAgent`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Coordinator Test: ContextCarry`** (1 nodes): `TestCreateProposalUsesTicketAndContextCarry`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Coordinator Test: Events`** (1 nodes): `TestCreateProposalRecordsEvent`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Coordinator Test: Unassigned Ticket`** (1 nodes): `TestCreateProposalRejectsUnassignedTicket`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `GetProjectInitDate()` and `Theme`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **Why does `main()` connect `Agent Exec Runner` to `Test Cases (Actions & Approvals)`, `App Test Infrastructure`, `Config Structs`, `Claude/Codex Agent Formatting`, `MCP Client Manager`, `App Commands & Initialization`, `TUI Components (Modal, Palette, Theme)`, `LLM Provider Factory`?**
  _High betweenness centrality (0.126) - this node is a cross-community bridge._
- **Why does `NewApp()` connect `App Commands & Initialization` to `Test Cases (Actions & Approvals)`, `Agent Exec Runner`, `Agent Detection`, `App Test Infrastructure`, `Claude/Codex Agent Formatting`, `TUI Components (Modal, Palette, Theme)`, `Kanban Board`, `Dashboard & Agent Utils`, `App Rendering (ANSI, Overlay)`?**
  _High betweenness centrality (0.074) - this node is a cross-community bridge._
- **Why does `newTestApp()` connect `App Test Infrastructure` to `Test Cases (Actions & Approvals)`, `Config Structs`, `Claude/Codex Agent Formatting`, `App Commands & Initialization`, `Dashboard & Agent Utils`?**
  _High betweenness centrality (0.057) - this node is a cross-community bridge._
- **Are the 34 inferred relationships involving `Open()` (e.g. with `main()` and `TestPaletteOpenClose()`) actually correct?**
  _`Open()` has 34 INFERRED edges - model-reasoned connections that need verification._
- **Are the 6 inferred relationships involving `newTestTicketView()` (e.g. with `Open()` and `.cleanup()`) actually correct?**
  _`newTestTicketView()` has 6 INFERRED edges - model-reasoned connections that need verification._
- **Are the 12 inferred relationships involving `newTestApp()` (e.g. with `TestAppTicketCreateShowsNotification()` and `TestAppTicketCreateNotificationDoesNotBlockNavigation()`) actually correct?**
  _`newTestApp()` has 12 INFERRED edges - model-reasoned connections that need verification._