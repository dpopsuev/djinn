package stubs

import "github.com/dpopsuev/djinn/uniform"

// UniformBuilder provides a fluent API for constructing test Uniforms.
type UniformBuilder struct {
	persona string
	roles   []string
	mode    string
	model   string
	prompt  string
	tools   []string
	reg     *uniform.RoleRegistry
	reqs    *uniform.ToolRequirements
}

// NewUniformBuilder creates a builder with defaults (developer role, agent mode).
func NewUniformBuilder() *UniformBuilder {
	return &UniformBuilder{
		persona: "test-agent",
		roles:   []string{"developer"},
		mode:    "agent",
		model:   "test-model",
		prompt:  "You are a test agent.",
		tools: []string{
			"Read", "Write", "Edit", "Bash", "Glob", "Grep",
			"git", "assignment", "discourse", "plan", "observe",
		},
		reg:  uniform.NewRoleRegistry(uniform.DefaultRoles()),
		reqs: uniform.DefaultToolRequirements(),
	}
}

// WithPersona sets the persona name.
func (b *UniformBuilder) WithPersona(name string) *UniformBuilder {
	b.persona = name
	return b
}

// WithRoles sets the role stack.
func (b *UniformBuilder) WithRoles(roles ...string) *UniformBuilder {
	b.roles = roles
	return b
}

// WithMode sets the operating mode.
func (b *UniformBuilder) WithMode(mode string) *UniformBuilder {
	b.mode = mode
	return b
}

// WithModel sets the LLM model.
func (b *UniformBuilder) WithModel(model string) *UniformBuilder {
	b.model = model
	return b
}

// WithPrompt sets the system prompt.
func (b *UniformBuilder) WithPrompt(prompt string) *UniformBuilder {
	b.prompt = prompt
	return b
}

// WithTools sets the available tool pool (before RBAC filtering).
func (b *UniformBuilder) WithTools(tools ...string) *UniformBuilder {
	b.tools = tools
	return b
}

// WithRegistry sets a custom role registry.
func (b *UniformBuilder) WithRegistry(reg *uniform.RoleRegistry) *UniformBuilder {
	b.reg = reg
	return b
}

// WithRequirements sets custom tool requirements.
func (b *UniformBuilder) WithRequirements(reqs *uniform.ToolRequirements) *UniformBuilder {
	b.reqs = reqs
	return b
}

// Build resolves the Uniform (persona → roles → capabilities → filtered tools).
func (b *UniformBuilder) Build() *uniform.Uniform {
	return uniform.NewUniform(
		b.persona,
		b.roles,
		b.reg,
		b.reqs,
		b.tools,
		b.mode,
		b.model,
		b.prompt,
	)
}

// GenSec returns a pre-configured GenSec Uniform.
func GenSec() *uniform.Uniform {
	return NewUniformBuilder().
		WithPersona("gensec").
		WithRoles("director", "manager").
		WithMode("plan").
		WithModel("opus").
		WithPrompt("You are the General Secretary.").
		Build()
}

// Coder returns a pre-configured Coder Uniform.
func Coder(id string) *uniform.Uniform {
	return NewUniformBuilder().
		WithPersona("coder-" + id).
		WithRoles("developer").
		WithMode("agent").
		WithModel("sonnet").
		WithPrompt("You are a Coder. Write code.").
		Build()
}

// SecondSecretary returns a pre-configured 2Sec Uniform.
func SecondSecretary() *uniform.Uniform {
	return NewUniformBuilder().
		WithPersona("2sec").
		WithRoles("manager").
		WithMode("plan").
		WithModel("sonnet").
		WithPrompt("You are the Second Secretary. Plan and schedule.").
		Build()
}
