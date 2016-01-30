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
    it "returns nil if no lines start with a space" do
      lines = [
        "",
        "// a comment",
        ".bad.suffix",
        "suffix",
        "a trailing space ",
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

  describe ".suffix_non_lowercase_index" do
    it "returns nil if all suffix rules are lowercase" do
      lines = [
        "",
        "foo",
        "// a comment",
        ".bad.suffix",
      ] 
      assert_equal nil, Linter.suffix_non_lowercase_index(lines)
    end

    it "returns the index if a suffix is not lower case" do
      lines = [
        "// a comment",
        "mixedCase",
      ] 
      assert_equal 1, Linter.suffix_non_lowercase_index(lines)

      lines = [
        "// a comment",
        "mixed.Case",
      ] 
      assert_equal 1, Linter.suffix_non_lowercase_index(lines)

      lines = [
        "// a comment",
        "mixed.caSe",
      ] 
      assert_equal 1, Linter.suffix_non_lowercase_index(lines)
    end

    it "ignores comments" do
      lines = [
        "// A comment",
        "suffix",
      ] 
      assert_equal nil, Linter.suffix_non_lowercase_index(lines)
    end
  end

  describe ".suffix_with_leading_dot_index" do
    it "returns nil no suffix rule starts with dot" do
      lines = [
        "",
        "foo",
        "// a comment",
      ] 
      assert_equal nil, Linter.suffix_with_leading_dot_index(lines)
    end

    it "returns the index if a suffix starts with dot" do
      lines = [
        "// a comment",
        ".foo",
      ] 
      assert_equal 1, Linter.suffix_with_leading_dot_index(lines)
    end

    # just to make sure we don't disable the "leading space" check by mistake
    it "strips leading spaces" do
      lines = [
        "// a comment",
        "foo",
        " .bar",
      ] 
      assert_equal 2, Linter.suffix_with_leading_dot_index(lines)
    end

    it "ignores comments" do
      lines = [
        "// .foo",
        "suffix",
      ] 
      assert_equal nil, Linter.suffix_with_leading_dot_index(lines)
    end
  end

  describe ".suffix_with_space_index" do
    it "returns nil no suffix rule contains a space" do
      lines = [
        "",
        " ",
        "foo",
        "// a comment",
      ] 
      assert_equal nil, Linter.suffix_with_space_index(lines)
    end

    it "returns the index if a suffix contains a space" do
      lines = [
        "// a comment",
        "foo. bar",
      ] 
      assert_equal 1, Linter.suffix_with_space_index(lines)

      lines = [
        "// a comment",
        "foo .bar",
      ] 
      assert_equal 1, Linter.suffix_with_space_index(lines)

      lines = [
        "// a comment",
        "foo.b ar",
      ] 
      assert_equal 1, Linter.suffix_with_space_index(lines)
    end

    # in theory, they are irrelevant, in practice we already have a test
    it "strips leading spaces" do
      lines = [
        "// a comment",
        " foo",
      ] 
      assert_equal nil, Linter.suffix_with_space_index(lines)
    end

    it "ignores trailing spaces" do
      lines = [
        "foo ",
        "bar",
      ] 
      assert_equal nil, Linter.suffix_with_space_index(lines)
    end

    it "ignores comments" do
      lines = [
        "// foo bar",
      ] 
      assert_equal nil, Linter.suffix_with_space_index(lines)
    end
  end

end
