require 'test_helper'

describe "Lint" do

  def self.list
    # I don't use File.readlines because it will preserve the \n at the end
    @lines ||= File.read(LIST_PATH).split("\n")
  end
  def list; self.class.list; end


  it "contains only lower-case suffixes" do
    index = list.find_index.each do |line|
      non_comment_line?(line) && 
      line =~ /[A-Z]/
    end
    assert_nil(index, -> { "List contains non-lowercase suffix at line #{index+1}: #{list[index]}" })
  end

  it "does not contain leading spaces" do
    index = Linter.leading_space_index(list)
    assert_nil(index, -> { "List contains a leading space at line #{index+1}: #{list[index]}" })
  end

  it "does not contain leading dots" do
    index = list.find_index.each do |line|
      # beginning of non-comment, followed by 0-more spaces, followed by leading .
      non_comment_line?(line) && 
      line =~ /\A\s*\./
    end
    assert_nil(index, -> { "List contains a leading dot at line #{index+1}: #{list[index]}" })
  end

  it "does not contain spaces in suffixes" do
    index = list.find_index.each do |line|
      suffix_line?(line) && 
      line =~ /([a-z0-9\-]+)(\s+)([a-z0-9\-]*)/
    end
    assert_nil(index, -> { "List contains a space in suffix at line #{index+1}: #{list[index]}" })
  end


  def suffix_line?(line)
    non_comment_line?(line) && non_blank_line?(line)
  end

  def non_comment_line?(line)
    !line.start_with?('//')
  end

  def non_blank_line?(line)
    line.strip != ""
  end

  def non_empty_line?(line)
    line != ""
  end

end
