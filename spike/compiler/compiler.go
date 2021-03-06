package compiler

import (
	"fmt"
	"sort"
	"spike-interpreter-go/spike/code"
	"spike-interpreter-go/spike/object"
	"spike-interpreter-go/spike/parser/ast"

	"github.com/pkg/errors"
)

type EmittedInstruction struct {
	Opcode   code.Opcode
	Position int
}

type CompilationScope struct {
	instructions        code.Instructions
	lastInstruction     EmittedInstruction
	previousInstruction EmittedInstruction
}

type Compiler struct {
	constants   []object.Object
	symbolTable *SymbolTable

	scopes     []CompilationScope
	scopeIndex int
}

func New() *Compiler {
	mainScope := CompilationScope{
		instructions:        code.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}

	symbolTable := NewSymbolTable()
	for i, builtin := range object.Builtins {
		symbolTable.DefineBuiltin(i, builtin.Name)
	}

	return &Compiler{
		constants:   []object.Object{},
		symbolTable: symbolTable,
		scopes:      []CompilationScope{mainScope},
		scopeIndex:  0,
	}
}

func NewWithState(symbolTable *SymbolTable, constants []object.Object) *Compiler {
	compiler := New()
	compiler.symbolTable = symbolTable
	compiler.constants = constants

	return compiler
}

func (compiler *Compiler) Compile(node ast.Node) error {
	switch node := node.(type) {
	case *ast.Program:
		for _, statement := range node.Statements {
			err := compiler.Compile(statement)
			if err != nil {
				return err
			}
		}

	case *ast.ExpressionStatement:
		err := compiler.Compile(node.Expression)
		if err != nil {
			return err
		}
		compiler.emit(code.OpPop)

	case *ast.BlockStatement:
		for _, statement := range node.Statements {
			err := compiler.Compile(statement)
			if err != nil {
				return err
			}
		}

	case *ast.InfixExpression:
		if node.Operator == "<" {
			err := compiler.Compile(node.Right)
			if err != nil {
				return err
			}

			err = compiler.Compile(node.Left)
			if err != nil {
				return err
			}

			compiler.emit(code.OpGreaterThan)

			return nil
		}

		err := compiler.Compile(node.Left)
		if err != nil {
			return err
		}

		err = compiler.Compile(node.Right)
		if err != nil {
			return err
		}

		switch node.Operator {
		case "+":
			compiler.emit(code.OpAdd)
		case "-":
			compiler.emit(code.OpSub)
		case "*":
			compiler.emit(code.OpMul)
		case "/":
			compiler.emit(code.OpDiv)
		case "==":
			compiler.emit(code.OpEqual)
		case "!=":
			compiler.emit(code.OpNotEqual)
		case ">":
			compiler.emit(code.OpGreaterThan)
		default:
			return fmt.Errorf("unknown operator: %s", node.Operator)
		}

	case *ast.PrefixExpression:
		err := compiler.Compile(node.Right)
		if err != nil {
			return err
		}

		switch node.Operator {
		case "!":
			compiler.emit(code.OpBang)
		case "-":
			compiler.emit(code.OpMinus)
		default:
			return errors.Errorf("invalid prefix operator: %s", node.Operator)
		}

	case *ast.Integer:
		integer := &object.Integer{Value: node.Value}
		compiler.emit(code.OpConstant, compiler.addConstant(integer))

	case *ast.String:
		str := &object.String{Value: node.Value}
		compiler.emit(code.OpConstant, compiler.addConstant(str))

	case *ast.Boolean:
		if node.Value {
			compiler.emit(code.OpTrue)
		} else {
			compiler.emit(code.OpFalse)
		}

	case *ast.IfExpression:
		err := compiler.Compile(node.Condition)
		if err != nil {
			return err
		}

		jumpNotTrueIndex := compiler.emit(code.OpJumpNotTrue, -1)

		err = compiler.Compile(node.Then)
		if err != nil {
			return err
		}

		if compiler.lastInstructionIs(code.OpPop) {
			compiler.removeLastInstruction()
		}

		if node.Else == nil {
			jumpIndex := compiler.emit(code.OpJump, -1)
			afterJumpIndex := len(compiler.scopes[compiler.scopeIndex].instructions)
			compiler.emit(code.OpNull)
			afterNullIndex := len(compiler.scopes[compiler.scopeIndex].instructions)

			compiler.changeOperand(jumpIndex, afterNullIndex)
			compiler.changeOperand(jumpNotTrueIndex, afterJumpIndex)
		} else {
			jumpIndex := compiler.emit(code.OpJump, -1)

			afterThenIndex := len(compiler.scopes[compiler.scopeIndex].instructions)
			compiler.changeOperand(jumpNotTrueIndex, afterThenIndex)

			err := compiler.Compile(node.Else)
			if err != nil {
				return err
			}

			if compiler.lastInstructionIs(code.OpPop) {
				compiler.removeLastInstruction()
			}

			afterElseIndex := len(compiler.scopes[compiler.scopeIndex].instructions)
			compiler.changeOperand(jumpIndex, afterElseIndex)
		}

	case *ast.LetStatement:
		symbol := compiler.symbolTable.Define(node.Name.Value)
		err := compiler.Compile(node.Value)
		if err != nil {
			return err
		}

		if symbol.SymbolScope == GlobalScope {
			compiler.emit(code.OpSetGlobal, symbol.Index)
		} else {
			compiler.emit(code.OpSetLocal, symbol.Index)
		}

	case *ast.Identifier:
		symbol, ok := compiler.symbolTable.Resolve(node.Value)
		if !ok {
			return errors.Errorf("unable to resolve identifier: %s", node.Value)
		}

		compiler.loadSymbol(symbol)

	case *ast.Array:
		for _, element := range node.Elements {
			err := compiler.Compile(element)
			if err != nil {
				return err
			}
		}

		compiler.emit(code.OpArray, len(node.Elements))

	case *ast.Hash:
		keys := make([]ast.Expression, 0)
		for key := range node.Pairs {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].String() < keys[j].String()
		})

		for _, key := range keys {
			err := compiler.Compile(key)
			if err != nil {
				return err
			}

			err = compiler.Compile(node.Pairs[key])
			if err != nil {
				return err
			}
		}

		compiler.emit(code.OpHash, len(node.Pairs)*2)

	case *ast.IndexExpression:
		err := compiler.Compile(node.Array)
		if err != nil {
			return err
		}

		err = compiler.Compile(node.Index)
		if err != nil {
			return err
		}

		compiler.emit(code.OpIndex)

	case *ast.FunctionExpression:
		compiler.enterScope()

		for _, parameter := range node.Parameters {
			compiler.symbolTable.Define(parameter.Value)
		}

		err := compiler.Compile(node.Body)
		if err != nil {
			return err
		}

		if compiler.lastInstructionIs(code.OpPop) {
			err = compiler.replaceLastPopWithReturn()
			if err != nil {
				return err
			}
		}

		if !compiler.lastInstructionIs(code.OpReturnValue) {
			compiler.emit(code.OpReturn)
		}

		freeSymbols := compiler.symbolTable.FreeSymbols
		localCount := compiler.symbolTable.numDefinitions
		instructions := compiler.leaveScope()

		for _, symbol := range freeSymbols {
			compiler.loadSymbol(symbol)
		}

		compiledFunction := &object.CompiledFunction{
			Instructions:    instructions,
			LocalsCount:     localCount,
			ParametersCount: len(node.Parameters),
		}
		index := compiler.addConstant(compiledFunction)
		compiler.emit(code.OpClosure, index, len(freeSymbols))

	case *ast.ReturnStatement:
		err := compiler.Compile(node.Result)
		if err != nil {
			return err
		}

		compiler.emit(code.OpReturnValue)

	case *ast.CallExpression:
		err := compiler.Compile(node.Function)
		if err != nil {
			return err
		}

		for _, argument := range node.Arguments {
			err = compiler.Compile(argument)
			if err != nil {
				return err
			}
		}

		compiler.emit(code.OpCall, len(node.Arguments))
	}

	return nil
}

func (compiler *Compiler) loadSymbol(symbol Symbol) {
	switch symbol.SymbolScope {
	case GlobalScope:
		compiler.emit(code.OpGetGlobal, symbol.Index)
	case BuiltinScope:
		compiler.emit(code.OpGetBuiltin, symbol.Index)
	case FreeScope:
		compiler.emit(code.OpGetFreeVar, symbol.Index)
	default:
		compiler.emit(code.OpGetLocal, symbol.Index)
	}
}

func (compiler *Compiler) addConstant(obj object.Object) int {
	compiler.constants = append(compiler.constants, obj)
	return len(compiler.constants) - 1
}

func (compiler *Compiler) emit(opcode code.Opcode, operands ...int) int {
	instruction, _ := code.Make(opcode, operands...)

	newInstructionIndex := len(compiler.scopes[compiler.scopeIndex].instructions)
	compiler.scopes[compiler.scopeIndex].instructions = append(compiler.scopes[compiler.scopeIndex].instructions, instruction...)

	compiler.scopes[compiler.scopeIndex].previousInstruction = compiler.scopes[compiler.scopeIndex].lastInstruction
	compiler.scopes[compiler.scopeIndex].lastInstruction = EmittedInstruction{
		Opcode:   opcode,
		Position: newInstructionIndex,
	}

	return newInstructionIndex
}

func (compiler *Compiler) removeLastInstruction() {
	compiler.scopes[compiler.scopeIndex].instructions = compiler.scopes[compiler.scopeIndex].instructions[:compiler.scopes[compiler.scopeIndex].lastInstruction.Position]
	compiler.scopes[compiler.scopeIndex].lastInstruction = compiler.scopes[compiler.scopeIndex].previousInstruction
}

func (compiler *Compiler) changeOperand(instructionIndex, operand int) {
	opcode := code.Opcode(compiler.scopes[compiler.scopeIndex].instructions[instructionIndex])
	newInstruction, _ := code.Make(opcode, operand)

	compiler.replaceInstruction(instructionIndex, newInstruction)
}

func (compiler *Compiler) Bytecode() *Bytecode {
	return &Bytecode{
		Instructions: compiler.scopes[compiler.scopeIndex].instructions,
		Constants:    compiler.constants,
	}
}

func (compiler *Compiler) replaceInstruction(instructionIndex int, instruction []byte) {
	for i := 0; i < len(instruction); i++ {
		compiler.scopes[compiler.scopeIndex].instructions[instructionIndex+i] = instruction[i]
	}
}

func (compiler *Compiler) lastInstructionIs(op code.Opcode) bool {
	return compiler.scopes[compiler.scopeIndex].lastInstruction.Opcode == op
}

func (compiler *Compiler) replaceLastPopWithReturn() error {
	lastPosition := compiler.scopes[compiler.scopeIndex].lastInstruction.Position

	returnInstruction, err := code.Make(code.OpReturnValue)
	if err != nil {
		return err
	}

	compiler.replaceInstruction(lastPosition, returnInstruction)
	compiler.scopes[compiler.scopeIndex].lastInstruction.Opcode = code.OpReturnValue

	return nil
}

func (compiler *Compiler) enterScope() {
	scope := CompilationScope{
		instructions:        code.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}

	compiler.symbolTable = NewEnclosedSymbolTable(compiler.symbolTable)
	compiler.scopes = append(compiler.scopes, scope)
	compiler.scopeIndex++
}

func (compiler *Compiler) leaveScope() code.Instructions {
	compiler.symbolTable = compiler.symbolTable.Outer
	instructions := compiler.scopes[compiler.scopeIndex].instructions
	compiler.scopes = compiler.scopes[:len(compiler.scopes)-1]
	compiler.scopeIndex--

	return instructions
}

type Bytecode struct {
	Instructions code.Instructions
	Constants    []object.Object
}
