require 'rubygems'
require 'bundler/setup'
require 'rake/testtask'

task :default => [:test]
Rake::TestTask.new do |t|
  t.libs << "test"
  t.test_files = FileList["test/*_test.rb"]
  t.verbose = true
end
