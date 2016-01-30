require 'test_helper'

describe Linter do

  describe ".empty_line?" do
    it "returns true if the line if empty" do
      [
        ["\n",      false],
        ["",        true ],
        [" ",       false],
        ["// foo",  false],
        [" // foo", false],
        ["foo",     false],
      ].each do |input, expected|
        assert_equal expected, Linter.empty_line?(input), "Expected .empty_line?(#{input.inspect}) => #{expected}"
        assert_equal !expected, Linter.non_empty_line?(input), "Expected .non_empty_line?(#{input.inspect}) => #{!expected}"
      end
    end
  end

  describe ".blank_line?" do
    it "returns true if the line contains only whitespaces" do
      [
        ["\n",      true ],
        ["",        true ],
        [" ",       true ],
        ["// foo",  false],
        [" // foo", false],
        ["foo",     false],
      ].each do |input, expected|
        assert_equal expected, Linter.blank_line?(input), "Expected .blank_line?(#{input.inspect}) => #{expected}"
        assert_equal !expected, Linter.non_blank_line?(input), "Expected .non_blank_line?(#{input.inspect}) => #{!expected}"
      end
    end
  end

  describe ".comment_line?" do
    it "returns true if the line starts with a comment" do
      [
        ["\n",      false],
        ["",        false],
        [" ",       false],
        ["// foo",  true ],
        [" // foo", true ],
        ["foo",     false],
      ].each do |input, expected|
        assert_equal expected, Linter.comment_line?(input), "Expected .comment_line?(#{input.inspect}) => #{expected}"
        assert_equal !expected, Linter.non_comment_line?(input), "Expected .non_comment_line?(#{input.inspect}) => #{!expected}"
      end
    end
  end

  describe ".suffix_line?" do
    it "returns true if the line if empty" do
      [
        ["\n",            false],
        ["",              false],
        [" ",             false],
        ["// foo",        false],
        [" // foo",       false],
        ["foo",           true ],
        [".foo",          true ],
        ["foo // bar",    true ],
        [" foo",          true ],
      ].each do |input, expected|
        assert_equal expected, Linter.suffix_line?(input), "Expected .suffix_line?(#{input.inspect}) => #{expected}"
        assert_equal !expected, Linter.non_suffix_line?(input), "Expected .non_suffix_line?(#{input.inspect}) => #{!expected}"
      end
    end
  end


  describe ".leading_space_index" do
    it "returns nil if the line does not start with a space" do
      lines = [
        "",
        "// a comment",
        ".bad.suffix",
        ".suffix",
        "a trailing space",
      ] 
      assert_equal nil, Linter.leading_space_index(lines)
    end

    it "returns the index if a line starts with a space" do
      lines = [
        "",
        " a leading space",
        "// a comment",
      ] 
      assert_equal 1, Linter.leading_space_index(lines)
    end
  end

end
