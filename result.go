package concurrent

import (
	"errors"
	"reflect"
)

var (
	errType = reflect.TypeOf((*error)(nil)).Elem()
)

type Result struct {
	values []reflect.Value
}

func (p Result) Err() (err error) {
	if len(p.values) == 0 {
		return
	}

	lastV := p.values[len(p.values)-1]
	if lastV.IsValid() && lastV.Type().ConvertibleTo(errType) && !lastV.IsNil() {
		err = lastV.Interface().(error)
		return
	}
	return
}

func (p Result) V(fn interface{}) error {
	return mapTo(fn, p.values)
}

func mapTo(fn interface{}, values []reflect.Value) (err error) {

	var valuesError error
	if len(values) > 0 {
		lastV := values[len(values)-1]
		if lastV.IsValid() && lastV.Type().ConvertibleTo(errType) && !lastV.IsNil() {
			valuesError = lastV.Interface().(error)
		}
	}

	if fn != nil {
		fnType := reflect.TypeOf(fn)
		if fnType.Kind() != reflect.Func {
			panic(errors.New("args of fn should be func, the fn args will receive the values result"))
		}

		if fnType.NumIn() > len(values) {
			if fnType.NumIn() == len(values)+1 {
				lastI := fnType.In(fnType.NumIn() - 1)
				if lastI.ConvertibleTo(errType) {
					var lastIV reflect.Value
					if len(values) > 0 && valuesError != nil {
						lastIV = reflect.ValueOf(valuesError)
					} else {
						lastIV = reflect.Zero(errType)
					}

					values = append(values, lastIV)
				}
			} else {
				panic(errors.New("fn args could not greater than result values"))
			}
		}

		fnValue := reflect.ValueOf(fn)
		retVales := fnValue.Call(values[0:fnType.NumIn()])

		if len(retVales) > 0 {
			lastV := retVales[len(retVales)-1]
			if lastV.IsValid() && !lastV.IsNil() && lastV.Type().ConvertibleTo(errType) {
				err = lastV.Interface().(error)
			}
		}
	}
	return
}
