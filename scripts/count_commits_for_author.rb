#!/usr/bin/env ruby

require 'json'
require 'pry'

def count_commits(json_file, author_name)
  data = JSON.parse File.read json_file
  shas = {}
  data.each do |event|
    sha = event['payload']['sha']
    event['payload']['contributors'].each do |contributor|
      next unless contributor['role'] == 'author' || contributor['role'] == 'co_author'
      next unless contributor['identity']['name'] == author_name
      shas[sha] = 1
    end
  end
  puts "#{author_name} has #{shas.length} distinct commit SHAs"
end

if ARGV.size < 2
  puts "Missing arguments: data.json 'author name'"
  exit(1)
end

count_commits(ARGV[0], ARGV[1])
