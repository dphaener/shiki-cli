#!/usr/bin/env ruby
# frozen_string_literal: true

require 'pathname'
require 'yaml'
require 'shellwords'

class MissionNotFoundError < StandardError; end

class FallbackMission
  attr_reader :path

  def initialize(mission_path)
    @path = Pathname.new(mission_path).realpath
    @config = nil
  end

  def load_config
    return @config unless @config.nil?

    config_path = @path / 'mission.yaml'
    unless config_path.exist?
      @config = {}
      return @config
    end

    begin
      data = YAML.safe_load(
        config_path.read,
        permitted_classes: [],
        permitted_symbols: [],
        aliases: true
      ) || {}
      @config = data.is_a?(Hash) ? data : {}
    rescue StandardError
      @config = {}
    end

    @config
  end

  def name
    load_config['name'] || @path.basename.to_s
  end

  def templates_dir
    @path / 'templates'
  end

  def commands_dir
    @path / 'commands'
  end

  def constitution_dir
    @path / 'constitution'
  end
end

def resolve_mission_path(project_root)
  sekkei_dir = project_root / '.sekkei'

  unless sekkei_dir.exist?
    raise MissionNotFoundError,
          "No .sekkei directory found in #{project_root}. " \
          'Is this a Sekkei project?'
  end

  active_link = sekkei_dir / 'active-mission'

  mission_path = if active_link.exist?
                   resolve_active_mission(active_link, sekkei_dir)
                 else
                   sekkei_dir / 'missions' / 'software-dev'
                 end

  return mission_path if mission_path.exist?

  # List available missions for error message
  missions_dir = sekkei_dir / 'missions'
  available = if missions_dir.exist?
                missions_dir.children
                            .select { |p| p.directory? && (p / 'mission.yaml').exist? }
                            .map { |p| p.basename.to_s }
                            .sort
              else
                []
              end

  raise MissionNotFoundError,
        "Active mission directory not found: #{mission_path}\n" \
        "Available missions: #{available.empty? ? 'none' : available.join(', ')}"
end

def resolve_active_mission(active_link, sekkei_dir)
  # Try resolving as symlink first
  if active_link.symlink?
    begin
      return active_link.realpath
    rescue StandardError
      # Fall through to other methods
    end
  end

  # Try reading as text file containing mission name
  if active_link.file?
    begin
      mission_name = active_link.read.strip
      unless mission_name.empty?
        candidate = sekkei_dir / 'missions' / mission_name
        return candidate if candidate.exist?
      end
    rescue StandardError
      # Fall through
    end
  end

  # Try OS-level readlink as last resort
  begin
    target = Pathname.new(File.readlink(active_link))
    candidate = (active_link.parent / target).realpath
    return candidate if candidate.exist?
  rescue StandardError
    # Fall through
  end

  # Default fallback
  sekkei_dir / 'missions' / 'software-dev'
end

def get_active_mission(project_root)
  FallbackMission.new(resolve_mission_path(project_root))
end

# Main execution
if ARGV.empty?
  warn '[sekkei] Error: Repository root path required as argument'
  exit 1
end

repo_root = Pathname.new(ARGV[0])

begin
  mission = get_active_mission(repo_root)
rescue MissionNotFoundError => e
  warn "[sekkei] Error: #{e.message}"
  exit 1
end

def emit(key, value)
  puts "#{key}=#{Shellwords.escape(value.to_s)}"
end

emit('MISSION_KEY', mission.path.basename)
emit('MISSION_PATH', mission.path)
emit('MISSION_NAME', mission.name)
emit('MISSION_TEMPLATES_DIR', mission.templates_dir)
emit('MISSION_COMMANDS_DIR', mission.commands_dir)
emit('MISSION_CONSTITUTION_DIR', mission.constitution_dir)

templates_dir = mission.templates_dir
emit('MISSION_SPEC_TEMPLATE', templates_dir / 'spec-template.md')
emit('MISSION_PLAN_TEMPLATE', templates_dir / 'plan-template.md')
emit('MISSION_TASKS_TEMPLATE', templates_dir / 'tasks-template.md')
emit('MISSION_TASK_PROMPT_TEMPLATE', templates_dir / 'task-template.md')
