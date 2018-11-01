package ui

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"unicode/utf8"

	spin "github.com/tj/go-spin"
	"github.com/ttacon/chalk"
	"github.com/xlab/treeprint"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	spinner = spin.New()
)

func init() {
	spinner.Set(spin.Spin1)
}

func Info(msg string, args ...interface{}) {
	fmt.Printf("%s %s\n", chalk.Bold.TextStyle(chalk.Blue.Color("==>")), chalk.Bold.TextStyle(fmt.Sprintf(msg, args...)))
}

func Verbose(msg string, args ...interface{}) {
	fmt.Println(chalk.Dim.TextStyle(fmt.Sprintf(msg, args...)))
}

func Success(msg string, args ...interface{}) {
	fmt.Printf("  %s %s\n", chalk.Bold.TextStyle(chalk.Green.Color("✔")), chalk.Bold.TextStyle(fmt.Sprintf(msg, args...)))
}

func Error(msg string, args ...interface{}) {
	fmt.Printf("  %s %s\n", chalk.Bold.TextStyle(chalk.Red.Color("✗")), chalk.Bold.TextStyle(fmt.Sprintf(msg, args...)))
}

func Fatal(msg string, args ...interface{}) {
	Error(msg, args...)
	os.Exit(1)
}

func Small(msg string) string {
	return chalk.Dim.TextStyle(msg)
}

func Emphasize(msg string) string {
	return chalk.Bold.TextStyle(chalk.Yellow.Color(msg))
}

func ConsoleWidth() int {
	width, _, err := terminal.GetSize(0)
	if err != nil || width <= 0 {
		return 80
	}
	return width
}

func Live(msg string) {
	// Format the message.
	msg = fmt.Sprintf("%s %s", spinner.Next(), strings.TrimSpace(msg))

	// Get the actual console width.
	lineLength := ConsoleWidth()

	// Shorten the message until it fits.
	for utf8.RuneCountInString(msg) > lineLength {
		msg = msg[0:len(msg)-4] + "…"
	}

	// Pad the message with spaces until it takes the entire line.
	// This is in order to clear the previous line.
	for utf8.RuneCountInString(msg) < lineLength {
		msg = msg + " "
	}

	fmt.Printf("%s\r", Small(msg))
}

func Tree(p string, ignore []string) error {
	root := treeprint.New()
	root.SetValue(p)
	if err := walk(p, root, ignore); err != nil {
		return err
	}
	Verbose(strings.TrimSpace(root.String()))
	return nil
}

func walk(p string, node treeprint.Tree, ignore []string) error {
	shouldIgnore := func(f os.FileInfo) bool {
		for _, i := range ignore {
			if f.Name() == i {
				return true
			}
		}
		return false
	}

	files, err := ioutil.ReadDir(p)
	if err != nil {
		return err
	}
	for _, file := range files {
		if shouldIgnore(file) {
			continue
		}
		if file.IsDir() {
			sub := node.AddBranch(file.Name())
			if err := walk(path.Join(p, file.Name()), sub, ignore); err != nil {
				return err
			}
			continue
		}

		node.AddNode(file.Name())
	}

	return nil
}
