require 'test_helper'
require 'public_suffix'

# This test runs against the current PSL file and ensures
# the definitions satisfies the test suite.
describe "PSL" do

  def self.tests
    File.readlines(File.join(ROOT, "test/tests.txt")).map do |line|
      line = line.strip
      next if line.empty?
      next if line.start_with?("//")
      input, output = line.split(", ")

      # handle the case of eval("null"), it must be eval("nil")
      input  = "nil" if input  == "null"
      output = "nil" if output == "null"

      input  = eval(input)
      output = eval(output)
      [input, output]
    end
  end


  # Parse the PSL and run the tests
  defs = File.read(LIST_PATH)
  PublicSuffix::List.default = PublicSuffix::List.parse(defs)

  self.tests.each do |input, output|
    it "test a rule" do
      domain = begin
        d = PublicSuffix.parse(input)
        [d.sld, d.tld].join(".")
      rescue
        nil
      end
      assert_equal(output, domain, "Expected `%s` -> `%s`" % [input, output])
    end
  end

end
