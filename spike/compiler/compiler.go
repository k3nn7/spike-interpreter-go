package compiler

import (
	"spike-interpreter-go/spike/code"
	"spike-interpreter-go/spike/eval/object"
	"spike-interpreter-go/spike/parser/ast"
)

type Compiler struct {
	instructions code.Instructions
	constants    []object.Object
}

func New() *Compiler {
	return &Compiler{
		instructions: code.Instructions{},
		constants:    []object.Object{},
	}
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
		return compiler.Compile(node.Expression)

	case *ast.InfixExpression:
		err := compiler.Compile(node.Left)
		if err != nil {
			return err
		}

		return compiler.Compile(node.Right)

	case *ast.Integer:
		integer := &object.Integer{Value: node.Value}
		compiler.emit(code.OpConstant, compiler.addConstant(integer))
	}

	return nil
}

func (compiler *Compiler) addConstant(obj object.Object) int {
	compiler.constants = append(compiler.constants, obj)
	return len(compiler.constants) - 1
}

func (compiler *Compiler) emit(opcode code.Opcode, operands ...int) int {
	instruction, _ := code.Make(opcode, operands...)

	newInstructionIndex := len(compiler.instructions)
	compiler.instructions = append(compiler.instructions, instruction...)
	return newInstructionIndex
}

func (compiler *Compiler) Bytecode() *Bytecode {
	return &Bytecode{
		Instructions: compiler.instructions,
		Constants:    compiler.constants,
	}
}

type Bytecode struct {
	Instructions code.Instructions
	Constants    []object.Object
}
