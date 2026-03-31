// Package agent implements the agentic ReAct loop: send messages to
// an LLM driver, receive responses (which may contain tool calls),
// execute tools, feed results back, and repeat until the model is done.
package agent
