---
agent1_name: "Architect"
agent1_role: "System architect"
agent1_system_prompt: "You are an experienced system architect. Design scalable, maintainable systems with clear justification for technical choices."
agent2_name: "Critic"
agent2_role: "Architecture critic"
agent2_system_prompt: "You critically analyze architecture proposals. Identify weaknesses, edge cases, and potential failures. Push for robust designs."
max_turns: 15
turn_timeout_seconds: 900

workspace_structure:
  - path: "diagrams/"
    type: "directory"
  - path: "designs/architecture.md"
    type: "file"
    content: "# System Architecture\n\n## Overview\n\n"
  - path: "designs/data-model.md"
    type: "file"
    content: "# Data Model\n\n"

completion_criteria: "Both agents approve a complete architecture design document with diagrams"

cost_limits:
  per_agent: 5.00
  total: 10.00
---

# Design Scalable E-Commerce Platform

Design the architecture for a high-traffic e-commerce platform.

Requirements:
- Handle 10,000 concurrent users
- 99.9% uptime SLA
- Global distribution (multi-region)
- Real-time inventory management
- Payment processing integration
- Search with autocomplete

Deliverable: Complete architecture document with:
- System diagram
- Data model
- Technology stack decisions with rationale
- Scalability strategy
- Failure modes and mitigations
