#!/usr/bin/env ruby
# frozen_string_literal: true

require 'pathname'

# Updates task checkbox status in tasks.md files
# Usage: task-status-updater.rb TASK_ID STATUS TASKS_FILE

if ARGV.length != 3
  warn 'Usage: task-status-updater.rb TASK_ID STATUS TASKS_FILE'
  exit 1
end

task_id = ARGV[0]
status = ARGV[1]
tasks_file = Pathname.new(ARGV[2])

unless tasks_file.exist?
  warn "Tasks file not found: #{tasks_file}"
  exit 1
end

# Read file content
begin
  text = tasks_file.read
rescue StandardError => e
  warn "Failed to read tasks file: #{e.message}"
  exit 1
end

# Build regex pattern to match task checkbox
# Pattern: ^(\s*-\s*)\[[ xX]\]\s+(TASK_ID)(\b.*)$
# Matches lines like: "  - [ ] WP01-description" or "  - [x] WP01-description"
escaped_task_id = Regexp.escape(task_id)
pattern = /^(\s*-\s*)\[[ xX]\]\s+(#{escaped_task_id})(\b.*)$/

# Determine checkbox state
checkbox = status == 'done' ? '[X]' : '[ ]'

# Replace first match only
match_count = 0
new_text = text.sub(pattern) do |_match|
  match_count += 1
  "#{Regexp.last_match(1)}#{checkbox} #{Regexp.last_match(2)}#{Regexp.last_match(3)}"
end

# Check if we found and replaced the task
if match_count.zero?
  warn "Task ID #{task_id} not found in #{tasks_file}"
  exit 1
end

# Write updated content back to file
begin
  tasks_file.write(new_text)
rescue StandardError => e
  warn "Failed to write tasks file: #{e.message}"
  exit 1
end
