package Benchmark

import "fmt"

func yellow(str string) string {
	return fmt.Sprintf("\u001b[33m%s\u001b[0m", str)
}

func green(str string) string {
	return fmt.Sprintf("\u001b[36m%s\u001b[0m", str)
}

func blue(str string) string {
	return fmt.Sprintf("\u001b[34m%s\u001b[0m", str)
}

func red(str string) string {
	return fmt.Sprintf("\u001b[31m%s\u001b[0m", str)
}
