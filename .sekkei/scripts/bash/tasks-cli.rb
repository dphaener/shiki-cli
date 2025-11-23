#!/usr/bin/env ruby
# frozen_string_literal: true

require 'pathname'
require 'yaml'
require 'time'
require 'fileutils'
require 'optparse'
require 'json'

# Sekkei Tasks CLI - Ruby implementation
# Manages work packages across kanban lanes: planned → doing → for_review → done

LANES = %w[planned doing for_review done].freeze
TIMESTAMP_FORMAT = '%Y-%m-%dT%H:%M:%SZ'

class TaskCliError < StandardError; end

class WorkPackage
  attr_reader :feature, :path, :current_lane, :relative_subpath
  attr_reader :frontmatter, :body, :padding

  def initialize(feature, path, current_lane, relative_subpath, frontmatter, body, padding)
    @feature = feature
    @path = Pathname.new(path)
    @current_lane = current_lane
    @relative_subpath = relative_subpath
    @frontmatter = frontmatter
    @body = body
    @padding = padding
  end

  def work_package_id
    extract_scalar('work_package_id') || extract_scalar('id')
  end

  def title
    extract_scalar('title')
  end

  def lane
    extract_scalar('lane') || current_lane
  end

  private

  def extract_scalar(key)
    match = @frontmatter.match(/^#{key}:\s*["']?([^"'\n]+)["']?/)
    match ? match[1].strip : nil
  end
end

module TasksSupport
  def self.find_repo_root(start_path = Dir.pwd)
    current = Pathname.new(start_path).realpath
    loop do
      return current if (current / '.git').exist? || (current / '.sekkei').exist?
      break if current.root?
      current = current.parent
    end
    raise TaskCliError, 'Not in a git repository or sekkei project'
  end

  def self.ensure_lane(value)
    normalized = value.to_s.downcase.tr('-', '_')
    raise TaskCliError, "Invalid lane: #{value}. Must be one of: #{LANES.join(', ')}" unless LANES.include?(normalized)
    normalized
  end

  def self.now_utc
    Time.now.utc.strftime(TIMESTAMP_FORMAT)
  end

  def self.split_frontmatter(text)
    # Match YAML frontmatter between --- markers
    if text.match(/\A(---\n.*?\n---\n)(\n*)(.*)\z/m)
      frontmatter = Regexp.last_match(1)
      padding = Regexp.last_match(2)
      body = Regexp.last_match(3)
      [frontmatter, body, padding]
    else
      ['', text, '']
    end
  end

  def self.build_document(frontmatter, body, padding)
    frontmatter + padding + body
  end

  def self.set_scalar(frontmatter, key, value)
    # Update existing key or add new one
    if frontmatter.match(/^#{key}:/)
      frontmatter.gsub(/^#{key}:.*$/, "#{key}: #{value}")
    else
      # Add before closing ---
      frontmatter.sub(/^---\n\z/, "#{key}: #{value}\n---\n")
    end
  end

  def self.append_activity_log(body, entry)
    activity_section = "## Activity Log\n\n"

    if body.include?('## Activity Log')
      # Append to existing log
      body.sub(/## Activity Log\n\n/) { |match| "#{match}#{entry}\n" }
    else
      # Create new activity log section
      "#{body}\n\n#{activity_section}#{entry}\n"
    end
  end

  def self.locate_work_package(repo_root, feature, wp_id)
    specs_dir = repo_root / 'sekkei-specs' / feature
    return nil unless specs_dir.exist?

    tasks_dir = specs_dir / 'tasks'
    return nil unless tasks_dir.exist?

    # Search all lanes for the work package
    LANES.each do |lane|
      lane_dir = tasks_dir / lane
      next unless lane_dir.exist?

      lane_dir.children.each do |file|
        next unless file.file? && file.extname == '.md'

        content = file.read
        frontmatter, body, padding = split_frontmatter(content)

        # Check if this is the work package we're looking for
        if content.match(/^(?:work_package_id|id):\s*["']?#{Regexp.escape(wp_id)}["']?/m)
          relative = file.relative_path_from(specs_dir)
          return WorkPackage.new(feature, file, lane, relative.to_s, frontmatter, body, padding)
        end
      end
    end

    nil
  end

  def self.git_status_clean?(repo_root)
    system('git', '-C', repo_root.to_s, 'diff', '--quiet', '--exit-code', out: File::NULL, err: File::NULL) &&
      system('git', '-C', repo_root.to_s, 'diff', '--cached', '--quiet', '--exit-code', out: File::NULL, err: File::NULL)
  end
end

# CLI Commands

def cmd_list(repo_root, feature, _options)
  specs_dir = repo_root / 'sekkei-specs' / feature
  tasks_dir = specs_dir / 'tasks'

  unless tasks_dir.exist?
    warn "No tasks directory found for feature #{feature}"
    exit 1
  end

  LANES.each do |lane|
    lane_dir = tasks_dir / lane
    puts "\n#{lane.upcase}:"

    if lane_dir.exist?
      files = lane_dir.children.select { |f| f.file? && f.extname == '.md' }.sort
      if files.empty?
        puts "  (empty)"
      else
        files.each do |file|
          puts "  - #{file.basename}"
        end
      end
    else
      puts "  (lane not created)"
    end
  end
end

def cmd_move(repo_root, feature, options)
  wp_id = options[:work_package_id]
  target_lane = TasksSupport.ensure_lane(options[:target_lane])
  note = options[:note] || "Moved to #{target_lane}"

  # Find the work package
  wp = TasksSupport.locate_work_package(repo_root, feature, wp_id)
  unless wp
    warn "Work package #{wp_id} not found in feature #{feature}"
    exit 1
  end

  if wp.current_lane == target_lane
    puts "Work package #{wp_id} already in #{target_lane} lane"
    exit 0
  end

  # Build new path
  specs_dir = repo_root / 'sekkei-specs' / feature
  new_path = specs_dir / 'tasks' / target_lane / wp.path.basename

  # Update frontmatter
  new_frontmatter = TasksSupport.set_scalar(wp.frontmatter, 'lane', target_lane)

  # Add activity log entry
  timestamp = TasksSupport.now_utc
  agent = ENV['USER'] || 'unknown'
  entry = "- **#{timestamp}** | #{agent} | #{wp.current_lane} → #{target_lane} | #{note}"
  new_body = TasksSupport.append_activity_log(wp.body, entry)

  # Write updated content
  new_content = TasksSupport.build_document(new_frontmatter, new_body, wp.padding)

  # Create target directory if needed
  new_path.parent.mkpath

  # Move file
  File.write(new_path, new_content)
  wp.path.unlink

  puts "Moved #{wp_id} from #{wp.current_lane} to #{target_lane}"
  puts "  #{wp.path} → #{new_path}"
end

def cmd_history(repo_root, feature, options)
  wp_id = options[:work_package_id]
  note = options[:note] || 'Activity recorded'

  wp = TasksSupport.locate_work_package(repo_root, feature, wp_id)
  unless wp
    warn "Work package #{wp_id} not found in feature #{feature}"
    exit 1
  end

  # Add activity log entry
  timestamp = TasksSupport.now_utc
  agent = ENV['USER'] || 'unknown'
  entry = "- **#{timestamp}** | #{agent} | #{wp.current_lane} | #{note}"
  new_body = TasksSupport.append_activity_log(wp.body, entry)

  # Write updated content
  new_content = TasksSupport.build_document(wp.frontmatter, new_body, wp.padding)
  File.write(wp.path, new_content)

  puts "Added activity log entry to #{wp_id}"
end

def cmd_rollback(_repo_root, _feature, _options)
  # Rollback would require maintaining move history
  # For now, implement as a placeholder
  warn 'Rollback functionality not yet implemented'
  warn 'To rollback a move, manually move the work package file back to the previous lane'
  exit 1
end

def cmd_accept(repo_root, feature, _options)
  specs_dir = repo_root / 'sekkei-specs' / feature
  tasks_dir = specs_dir / 'tasks'

  unless tasks_dir.exist?
    warn "No tasks directory found for feature #{feature}"
    exit 1
  end

  # Check if all work packages are in 'done'
  incomplete_count = 0

  LANES.reject { |l| l == 'done' }.each do |lane|
    lane_dir = tasks_dir / lane
    next unless lane_dir.exist?

    files = lane_dir.children.select { |f| f.file? && f.extname == '.md' }
    incomplete_count += files.size

    files.each do |file|
      puts "  ⚠️  #{lane}/#{file.basename} - not done"
    end
  end

  if incomplete_count.zero?
    puts "✅ All work packages are complete (in 'done' lane)"
    puts "Feature #{feature} is ready for acceptance"
    exit 0
  else
    warn "❌ #{incomplete_count} work package(s) not yet complete"
    warn "Move all work packages to 'done' lane before accepting"
    exit 1
  end
end

def cmd_merge(repo_root, feature, options)
  # Merge is about git branch merging, not work packages
  # This would typically call git commands
  strategy = options[:strategy] || 'merge'
  target_branch = options[:target] || 'main'

  unless TasksSupport.git_status_clean?(repo_root)
    warn 'Working directory has uncommitted changes'
    warn 'Commit or stash changes before merging'
    exit 1
  end

  puts "Merging feature #{feature} into #{target_branch} using #{strategy} strategy..."

  # This is a simplified implementation
  # Full implementation would handle worktrees, pushes, cleanup, etc.
  warn 'Full merge functionality should be implemented based on project requirements'
  exit 1
end

# Main CLI

def main
  command = ARGV.shift

  unless command
    warn 'Usage: tasks-cli.rb <command> [options]'
    warn 'Commands: list, move, history, rollback, accept, merge'
    exit 1
  end

  options = {}
  repo_root = TasksSupport.find_repo_root

  # Get current branch/feature
  feature = `git rev-parse --abbrev-ref HEAD 2>/dev/null`.strip
  if feature.empty? || feature == 'HEAD'
    # Fallback: detect from current directory
    current_dir = Pathname.pwd
    if current_dir.to_s.include?('sekkei-specs')
      feature = current_dir.to_s.match(/sekkei-specs\/([^\/]+)/)[1]
    else
      warn 'Cannot determine current feature. Run from a feature branch or sekkei-specs directory.'
      exit 1
    end
  end

  # Parse command-specific arguments
  OptionParser.new do |opts|
    opts.banner = "Usage: tasks-cli.rb #{command} [options]"

    case command
    when 'move'
      opts.on('--wp ID', '--work-package ID', 'Work package ID') { |v| options[:work_package_id] = v }
      opts.on('--to LANE', '--target LANE', 'Target lane') { |v| options[:target_lane] = v }
      opts.on('--note NOTE', 'Activity note') { |v| options[:note] = v }
    when 'history'
      opts.on('--wp ID', '--work-package ID', 'Work package ID') { |v| options[:work_package_id] = v }
      opts.on('--note NOTE', 'Activity note') { |v| options[:note] = v }
    when 'merge'
      opts.on('--strategy STRATEGY', 'Merge strategy') { |v| options[:strategy] = v }
      opts.on('--target BRANCH', 'Target branch') { |v| options[:target] = v }
    end
  end.parse!

  # Execute command
  case command
  when 'list'
    cmd_list(repo_root, feature, options)
  when 'move'
    cmd_move(repo_root, feature, options)
  when 'history'
    cmd_history(repo_root, feature, options)
  when 'rollback'
    cmd_rollback(repo_root, feature, options)
  when 'accept'
    cmd_accept(repo_root, feature, options)
  when 'merge'
    cmd_merge(repo_root, feature, options)
  else
    warn "Unknown command: #{command}"
    warn 'Commands: list, move, history, rollback, accept, merge'
    exit 1
  end
rescue TaskCliError => e
  warn "Error: #{e.message}"
  exit 1
end

main if __FILE__ == $PROGRAM_NAME
