package vm

import (
	"encoding/binary"
	"spike-interpreter-go/spike/code"
	"spike-interpreter-go/spike/compiler"
	"spike-interpreter-go/spike/eval/object"

	"github.com/pkg/errors"
)

const StackSize = 2048

var (
	True  = &object.Boolean{Value: true}
	False = &object.Boolean{Value: false}
)

type VM struct {
	constants    []object.Object
	instructions code.Instructions

	stack []object.Object
	sp    int
}

func New(bytecode *compiler.Bytecode) *VM {
	return &VM{
		constants:    bytecode.Constants,
		instructions: bytecode.Instructions,
		stack:        make([]object.Object, StackSize),
		sp:           0,
	}
}

func (vm *VM) Run() error {
	for ip := 0; ip < len(vm.instructions); ip++ {
		op := code.Opcode(vm.instructions[ip])

		switch op {
		case code.OpConstant:
			index := binary.BigEndian.Uint16(vm.instructions[ip+1:])
			ip += 2

			err := vm.push(vm.constants[index])
			if err != nil {
				return err

			}

		case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv:
			err := vm.executeBinaryIntegerOperation(op)
			if err != nil {
				return err
			}

		case code.OpEqual, code.OpNotEqual, code.OpGreaterThan:
			err := vm.executeComparison(op)
			if err != nil {
				return err
			}

		case code.OpTrue:
			err := vm.push(True)
			if err != nil {
				return err
			}

		case code.OpFalse:
			err := vm.push(False)
			if err != nil {
				return err
			}

		case code.OpPop:
			vm.pop()
		}
	}
	return nil
}

func (vm *VM) executeBinaryIntegerOperation(opcode code.Opcode) error {
	right := vm.pop()
	left := vm.pop()
	leftValue := left.(*object.Integer).Value
	rightValue := right.(*object.Integer).Value

	var result int64
	switch opcode {
	case code.OpAdd:
		result = leftValue + rightValue
	case code.OpSub:
		result = leftValue - rightValue
	case code.OpMul:
		result = leftValue * rightValue
	case code.OpDiv:
		result = leftValue / rightValue
	}
	return vm.push(&object.Integer{Value: result})
}

func (vm *VM) executeComparison(op code.Opcode) error {
	right := vm.pop()
	left := vm.pop()

	if right.Type() != left.Type() {
		return errors.Errorf("both operands must have same type, had: %s and %s", left.Type(), right.Type())
	}

	if right.Type() == object.IntegerType {
		return vm.executeIntegerComparison(left, right, op)
	}

	if right.Type() == object.BooleanType {
		return vm.executeBooleanComparison(left, right, op)
	}

	return errors.Errorf("unable to compare variables of type %s and %s", left.Type(), right.Type())
}

func (vm *VM) executeIntegerComparison(left object.Object, right object.Object, op code.Opcode) error {
	leftInt := left.(*object.Integer).Value
	rightInt := right.(*object.Integer).Value

	switch op {
	case code.OpEqual:
		return vm.push(nativeBoolToBoolean(leftInt == rightInt))
	case code.OpNotEqual:
		return vm.push(nativeBoolToBoolean(leftInt != rightInt))
	case code.OpGreaterThan:
		return vm.push(nativeBoolToBoolean(leftInt > rightInt))
	}

	return errors.Errorf("unexpected operation: %d", op)
}

func (vm *VM) executeBooleanComparison(left object.Object, right object.Object, op code.Opcode) error {
	leftBool := left.(*object.Boolean).Value
	rightBool := right.(*object.Boolean).Value

	switch op {
	case code.OpEqual:
		return vm.push(nativeBoolToBoolean(leftBool == rightBool))
	case code.OpNotEqual:
		return vm.push(nativeBoolToBoolean(leftBool != rightBool))
	}

	return errors.Errorf("unexpected operation: %d", op)
}

func nativeBoolToBoolean(nativeBool bool) object.Object {
	if nativeBool {
		return True
	} else {
		return False
	}
}

func (vm *VM) LastPoppedStackElement() object.Object {
	return vm.stack[vm.sp]
}

func (vm *VM) push(o object.Object) error {
	if vm.sp >= StackSize {
		return errors.New("stack overflow")
	}

	vm.stack[vm.sp] = o
	vm.sp++

	return nil
}

func (vm *VM) pop() object.Object {
	result := vm.stack[vm.sp-1]
	vm.sp--
	return result
}
