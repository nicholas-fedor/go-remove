// Package main provides the entry point for the go-remove command-line tool.
package main

import "github.com/nicholas-fedor/go-remove/cmd"

// main runs the go-remove command by invoking the root command execution.
func main() {
	// Delegate to cmd.Execute for CLI parsing and execution, handling errors
	// via stderr and exit codes as needed.
	cmd.Execute()
}
