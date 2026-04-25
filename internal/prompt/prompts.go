package prompt

import "fmt"

func GenerateProposal(ticketID, title, description, agent, contextCarry string) string {
	return fmt.Sprintf(
		"You are preparing a worker prompt for an AI coding agent.\n\n"+
			"Ticket ID: %s\nTitle: %s\nDescription: %s\nAssigned agent: %s\n"+
			"Context from previous runs: %s\n\n"+
			"Return only the final worker prompt that the assigned agent should execute.\n"+
			"Do not include chain-of-thought, reasoning, analysis, `<think>` tags, commentary, or markdown fences.\n"+
			"Use the exact ticket ID above and do not invent or substitute another ticket ID.\n"+
			"Include all relevant context and specific instructions in plain text.",
		ticketID,
		title,
		description,
		agent,
		contextCarry,
	)
}

func SummarizeContext(ticketID, outcome, summary string) string {
	return fmt.Sprintf(
		"You are summarizing a completed agent run for future context carry.\n\n"+
			"Ticket ID: %s\nOutcome: %s\nWorker summary: %s\n\n"+
			"Return a compact resumable context summary that the next agent run can pick up from. "+
			"Focus on what was done, what remains, and any important decisions made.",
		ticketID,
		outcome,
		summary,
	)
}
