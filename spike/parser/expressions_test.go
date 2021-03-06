package parser

import (
	"spike-interpreter-go/spike/lexer"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_prefix_expressions(t *testing.T) {
	testCases := map[string]struct {
		input           string
		expectedProgram string
	}{
		"single identifier": {
			input:           "foobar;",
			expectedProgram: "foobar\n",
		},
		"single integer": {
			input:           "10;",
			expectedProgram: "10\n",
		},
		"true keyword": {
			input:           "true;",
			expectedProgram: "true\n",
		},
		"false keyword": {
			input:           "false;",
			expectedProgram: "false\n",
		},
		"let statement with two identifiers": {
			input:           "let var1 = var2;",
			expectedProgram: "let var1 = var2\n",
		},
		"let statement with integer literal": {
			input:           "let var = 125;",
			expectedProgram: "let var = 125\n",
		},
		"return statement with integer literal": {
			input:           "return 7;",
			expectedProgram: "return 7\n",
		},
		"return statement with identifier": {
			input:           "return result;",
			expectedProgram: "return result\n",
		},
		"not identifier": {
			input:           "! boolVariable;",
			expectedProgram: "(!boolVariable)\n",
		},
		"not integer": {
			input:           "! 0;",
			expectedProgram: "(!0)\n",
		},
		"negate integer": {
			input:           "- 10;",
			expectedProgram: "(-10)\n",
		},
		"negate identifier": {
			input:           "- variable;",
			expectedProgram: "(-variable)\n",
		},
		"negate boolean": {
			input:           "return !false;",
			expectedProgram: "return (!false)\n",
		},
	}

	for testCaseName, testCase := range testCases {
		t.Run(testCaseName, func(t *testing.T) {
			program, err := New(lexer.New(strings.NewReader(testCase.input))).ParseProgram()

			assert.NoError(t, err)
			assert.Equal(t, testCase.expectedProgram, program.String())
		})
	}
}

func Test_infix_expressions(t *testing.T) {
	testCases := map[string]struct {
		input           string
		expectedProgram string
	}{
		"add two integers": {
			input:           "5 + 5;",
			expectedProgram: "(5 + 5)\n",
		},
		"multiply two integers": {
			input:           "5 * 5;",
			expectedProgram: "(5 * 5)\n",
		},
		"add and multiply": {
			input:           "5 + 5 * 5;",
			expectedProgram: "(5 + (5 * 5))\n",
		},
		"multiply and add": {
			input:           "5 * 5 + 5;",
			expectedProgram: "((5 * 5) + 5)\n",
		},
		"three additions": {
			input:           "1 + 2 + 3;",
			expectedProgram: "((1 + 2) + 3)\n",
		},
		"subtraction": {
			input:           "2 - 3;",
			expectedProgram: "(2 - 3)\n",
		},
		"division": {
			input:           "2 / 3;",
			expectedProgram: "(2 / 3)\n",
		},
		"equation": {
			input:           "2 + 3 * 5 - 8 / 15;",
			expectedProgram: "((2 + (3 * 5)) - (8 / 15))\n",
		},
		"boolean expression": {
			input:           "2 > 3 || 3 < 2 && 2 == 2 || 2 != 3 && 3 >= 2 == 5 <= 4;",
			expectedProgram: "(((2 > 3) || ((3 < 2) && (2 == 2))) || ((2 != 3) && ((3 >= (2 == 5)) <= 4)))\n",
		},
		"grouped expressions": {
			input:           "(2 + 2) * 3;",
			expectedProgram: "((2 + 2) * 3)\n",
		},
	}

	for testCaseName, testCase := range testCases {
		t.Run(testCaseName, func(t *testing.T) {
			program, err := New(lexer.New(strings.NewReader(testCase.input))).ParseProgram()

			assert.NoError(t, err)
			assert.Equal(t, testCase.expectedProgram, program.String())
		})
	}
}

func Test_invalid_expressions(t *testing.T) {
	testCases := map[string]struct {
		code          string
		expectedError string
	}{
		"let after minus operator": {
			code:          `-let;`,
			expectedError: `"let" is not a valid prefix expression`,
		},
		"return after minus operator": {
			code:          `-return;`,
			expectedError: `"return" is not a valid prefix expression`,
		},
	}

	for testCaseName, testCase := range testCases {
		t.Run(testCaseName, func(t *testing.T) {
			parser := New(lexer.New(strings.NewReader(testCase.code)))

			_, err := parser.ParseProgram()

			assert.EqualError(t, err, testCase.expectedError)
		})
	}
}
