package main

import "fmt"

func foo() string {
	panic("foo")
	return ""
}

func endpoint(
	procedure func() string,
) func() string {
	return func() (result string) {
		finnaly := func() {
			e := recover()
			if e == nil {
				fmt.Printf("OK\n")
			} else {
				result = "SomethingWentWrong"
			}
		}
		defer finnaly()
		result = procedure()
		return result
	}
}

func main() {
	fooEndpoint := endpoint(foo)
	output := fooEndpoint()
	fmt.Printf("Output %s\n", output)
}