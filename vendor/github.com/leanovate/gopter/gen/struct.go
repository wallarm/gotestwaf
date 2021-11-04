package gen

import (
	"reflect"

	"github.com/leanovate/gopter"
)

// Struct generates a given struct type.
// rt has to be the reflect type of the struct, gens contains a map of field generators.
// Note that the result types of the generators in gen have to match the type of the correspoinding
// field in the struct. Also note that only public fields of a struct can be generated
func Struct(rt reflect.Type, gens map[string]gopter.Gen) gopter.Gen {
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	if rt.Kind() != reflect.Struct {
		return Fail(rt)
	}
	fieldGens := []gopter.Gen{}
	fieldTypes := []reflect.Type{}
	assignable := reflect.New(rt).Elem()
	for i := 0; i < rt.NumField(); i++ {
		fieldName := rt.Field(i).Name
		if !assignable.Field(i).CanSet() {
			continue
		}

		gen := gens[fieldName]
		if gen != nil {
			fieldGens = append(fieldGens, gen)
			fieldTypes = append(fieldTypes, rt.Field(i).Type)
		}
	}

	buildStructType := reflect.FuncOf(fieldTypes, []reflect.Type{rt}, false)
	unbuildStructType := reflect.FuncOf([]reflect.Type{rt}, fieldTypes, false)

	buildStructFunc := reflect.MakeFunc(buildStructType, func(args []reflect.Value) []reflect.Value {
		result := reflect.New(rt)
		for i := 0; i < rt.NumField(); i++ {
			if _, ok := gens[rt.Field(i).Name]; !ok {
				continue
			}
			if !assignable.Field(i).CanSet() {
				continue
			}
			result.Elem().Field(i).Set(args[0])
			args = args[1:]
		}
		return []reflect.Value{result.Elem()}
	})
	unbuildStructFunc := reflect.MakeFunc(unbuildStructType, func(args []reflect.Value) []reflect.Value {
		s := args[0]
		results := []reflect.Value{}
		for i := 0; i < s.NumField(); i++ {
			if _, ok := gens[rt.Field(i).Name]; !ok {
				continue
			}
			if !assignable.Field(i).CanSet() {
				continue
			}
			results = append(results, s.Field(i))
		}
		return results
	})

	return gopter.DeriveGen(
		buildStructFunc.Interface(),
		unbuildStructFunc.Interface(),
		fieldGens...,
	)
}

// StructPtr generates pointers to a given struct type.
// Note that StructPtr does not generate nil, if you want to include nil in your
// testing you should combine gen.PtrOf with gen.Struct.
// rt has to be the reflect type of the struct, gens contains a map of field generators.
// Note that the result types of the generators in gen have to match the type of the correspoinding
// field in the struct. Also note that only public fields of a struct can be generated
func StructPtr(rt reflect.Type, gens map[string]gopter.Gen) gopter.Gen {
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	buildPtrType := reflect.FuncOf([]reflect.Type{rt}, []reflect.Type{reflect.PtrTo(rt)}, false)
	unbuildPtrType := reflect.FuncOf([]reflect.Type{reflect.PtrTo(rt)}, []reflect.Type{rt}, false)

	buildPtrFunc := reflect.MakeFunc(buildPtrType, func(args []reflect.Value) []reflect.Value {
		sp := reflect.New(rt)
		sp.Elem().Set(args[0])
		return []reflect.Value{sp}
	})
	unbuildPtrFunc := reflect.MakeFunc(unbuildPtrType, func(args []reflect.Value) []reflect.Value {
		return []reflect.Value{args[0].Elem()}
	})

	return gopter.DeriveGen(
		buildPtrFunc.Interface(),
		unbuildPtrFunc.Interface(),
		Struct(rt, gens),
	)
}
