---
agent1_name: "Developer"
agent1_role: "Implement feature"
agent1_system_prompt: "You write clean, well-documented code. Focus on simplicity and testability."
agent2_name: "Reviewer"
agent2_role: "Code reviewer"
agent2_system_prompt: "You review code for correctness, style, and best practices. Provide constructive feedback."
max_turns: 10
turn_timeout_seconds: 600

workspace_structure:
  - path: "src/"
    type: "directory"
  - path: "tests/"
    type: "directory"
  - path: "README.md"
    type: "file"
    content: "# Feature Implementation\n\n"

cost_limits:
  per_agent: 2.00
  total: 4.00
---

# Code Review Collaboration

Implement a simple user authentication feature.

Requirements:
- Password hashing with bcrypt
- Session token generation
- Login and logout endpoints
- Unit tests with >80% coverage

Deliverable: Working code with passing tests and reviewer approval.
