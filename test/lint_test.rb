require 'test_helper'

describe "Lint" do

  def self.list
    # I don't use File.readlines because it will preserve the \n at the end
    @lines ||= File.read(LIST_PATH).split("\n")
  end
  def list; self.class.list; end


  it "does not contain leading spaces" do
    index = Linter.leading_space_index(list)
    assert_nil(index, -> { "List contains a leading space at line #{index+1}: #{list[index]}" })
  end

  it "contains only lower-case suffixes" do
    index = Linter.suffix_non_lowercase_index(list)
    assert_nil(index, -> { "List contains non-lowercase suffix at line #{index+1}: #{list[index]}" })
  end

  it "does not contain suffix with leading dots" do
    index = Linter.suffix_with_leading_dot_index(list)
    assert_nil(index, -> { "List contains a leading dot at line #{index+1}: #{list[index]}" })
  end

  it "does not contain spaces in suffixes" do
    index = Linter.suffix_with_space_index(list)
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
