require 'rubygems'
require 'bundler/setup'
require 'minitest/autorun'
require 'minitest/reporters'

Minitest::Reporters.use! Minitest::Reporters::DefaultReporter.new(:color => true)

ROOT          = File.expand_path('../../', __FILE__)
LIST_FILENAME = 'public_suffix_list.dat'
LIST_PATH     = File.join(ROOT, LIST_FILENAME)
