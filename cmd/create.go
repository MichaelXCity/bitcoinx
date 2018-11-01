package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/blocklayerhq/chainkit/pkg/builder"
	"github.com/blocklayerhq/chainkit/pkg/httpfs"
	"github.com/blocklayerhq/chainkit/pkg/ui"
	"github.com/blocklayerhq/chainkit/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type templateContext struct {
	Name    string
	RootDir string
	GoPkg   string
}

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create an application",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		rootDir := path.Join(getCwd(cmd), name)
		create(name, rootDir)
	},
}

func init() {
	createCmd.Flags().String("cwd", ".", "specifies the current working directory")

	rootCmd.AddCommand(createCmd)
}

func create(name, rootDir string) {
	ctx := context.Background()

	ui.Info("Creating a new blockchain app in %s", ui.Emphasize(rootDir))

	if err := scaffold(name, rootDir); err != nil {
		ui.Fatal("Failed to initialize: %v", err)
	}

	b := builder.New(rootDir, name)
	if err := b.Build(ctx, builder.BuildOpts{}); err != nil {
		ui.Fatal("Failed to build the application: %v", err)
	}

	ui.Success("Success! Created %s at %s", ui.Emphasize(name), ui.Emphasize(rootDir))
	printGettingStarted(name)
}

func printGettingStarted(name string) {
	fmt.Printf(`
Inside that directory, you can run several commands:

  %s
    Starts the application.

  %s
    Build the application.

We suggest that you begin by typing:
  %s %s
  %s
`,
		ui.Emphasize("chainkit start"),
		ui.Emphasize("chainkit build"),
		ui.Emphasize("cd"),
		name,
		ui.Emphasize("chainkit start"),
	)
}

func scaffold(name, rootDir string) error {
	ui.Info("Scaffolding base application")

	gosource := goSrc()

	if !strings.HasPrefix(rootDir, gosource) {
		return fmt.Errorf("you must run this command within your GOPATH (%q)", goPath())
	}

	// Make sure the destination path doesn't exist.
	if _, err := os.Stat(rootDir); !os.IsNotExist(err) {
		return fmt.Errorf("destination path %q already exists", rootDir)
	}

	ctx := &templateContext{
		Name:    name,
		RootDir: rootDir,
		GoPkg:   strings.TrimPrefix(rootDir, gosource+"/"),
	}

	if err := extractFiles(ctx, rootDir); err != nil {
		return err
	}
	if err := ui.Tree(rootDir, []string{"k8s"}); err != nil {
		return err
	}

	return nil
}

func extractFiles(ctx *templateContext, dest string) error {
	err := httpfs.Walk(templates.Assets, "/", func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return extractFile(ctx, path, dest, fi)
	})
	return err
}

func extractFile(ctx *templateContext, src, dst string, fi os.FileInfo) error {
	// Templatize the file name.
	parsedSrc, err := templatize(ctx, src, src)
	if err != nil {
		return err
	}

	dstPath := path.Join(dst, string(parsedSrc))
	if fi.IsDir() {
		return os.MkdirAll(dstPath, fi.Mode())
	}

	data, err := httpfs.ReadFile(templates.Assets, src)
	if err != nil {
		return errors.Wrap(err, "unable to read template file")
	}

	// Handle templates
	if filepath.Ext(dstPath) == ".tmpl" {
		// Parse template
		data, err = templatize(ctx, dstPath, string(data))
		if err != nil {
			return errors.Wrap(err, "unable to templetaize")
		}

		// Remove .tpl from the file path
		dstPath = strings.TrimSuffix(dstPath, ".tmpl")

	}

	if err := ioutil.WriteFile(dstPath, data, fi.Mode()); err != nil {
		return errors.Wrap(err, "unable to write to destination")
	}

	return nil
}

func templatize(ctx *templateContext, name, input string) ([]byte, error) {
	t, err := template.New(name).Parse(input)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, ctx); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
