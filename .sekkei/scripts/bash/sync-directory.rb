#!/usr/bin/env ruby
# frozen_string_literal: true

require 'pathname'
require 'fileutils'

# Synchronizes directories by copying source to destination
# Removes destination if it exists, then copies source recursively
# Filters out __pycache__ directories
# Usage: sync-directory.rb SOURCE_DIR DEST_DIR

if ARGV.length != 2
  warn 'Usage: sync-directory.rb SOURCE_DIR DEST_DIR'
  exit 1
end

source_dir = Pathname.new(ARGV[0])
dest_dir = Pathname.new(ARGV[1])

unless source_dir.exist?
  warn "Source directory not found: #{source_dir}"
  exit 1
end

unless source_dir.directory?
  warn "Source path is not a directory: #{source_dir}"
  exit 1
end

# Remove destination if it exists (full replacement)
if dest_dir.exist?
  begin
    FileUtils.rm_rf(dest_dir)
  rescue StandardError => e
    warn "Failed to remove existing destination: #{e.message}"
    exit 1
  end
end

# Copy directory tree, filtering out __pycache__
def copy_tree_filtered(src, dst)
  FileUtils.mkdir_p(dst)

  Pathname.new(src).children.each do |child|
    # Skip __pycache__ directories
    next if child.basename.to_s == '__pycache__'

    dest_path = Pathname.new(dst) / child.basename

    if child.directory?
      copy_tree_filtered(child, dest_path)
    else
      FileUtils.cp(child, dest_path, preserve: true)
    end
  end
end

begin
  copy_tree_filtered(source_dir, dest_dir)
rescue StandardError => e
  warn "Failed to copy directory tree: #{e.message}"
  exit 1
end
