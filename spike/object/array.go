package object

import "strings"

type Array struct {
	Elements []Object
}

func (array *Array) Type() ObjectType {
	return ArrayType
}

func (array *Array) Inspect() string {
	out := strings.Builder{}

	out.WriteString("[")
	for i, element := range array.Elements {
		out.WriteString(element.Inspect())
		if i < len(array.Elements)-1 {
			out.WriteString(", ")
		}
	}
	out.WriteString("]")

	return out.String()
}

func (array *Array) Equal(other Object) bool {
	otherArray, ok := other.(*Array)
	if !ok {
		return false
	}

	if len(array.Elements) != len(otherArray.Elements) {
		return false
	}

	for i := range array.Elements {
		if !array.Elements[i].Equal(otherArray.Elements[i]) {
			return false
		}
	}

	return true
}
