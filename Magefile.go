//go:build mage

package main

import (
	"embed"
	"errors"
	"io/fs"
	"os"
	"strings"

	"github.com/magefile/mage/sh"
)

//go:embed *.proto testproto
var files embed.FS

// Gen (re-)generates auto-generated code.
func Gen() error {
	var fileArgs []string
	if err := fs.WalkDir(files, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(p, ".pb.go") {
			if r := os.Remove(p); r != nil {
				return r
			}
		}
		if !strings.HasSuffix(p, ".proto") {
			return nil
		}

		fileArgs = append(fileArgs, p)
		return nil
	}); err != nil {
		return err
	}

	if r := os.Mkdir("goproto", 0755); r != nil && !errors.Is(r, os.ErrExist) {
		return r
	}
	args := append([]string{
		// "-I=.",
		"--go_out=.", "--go_opt=paths=source_relative",
		"--go-grpc_out=.", "--go-grpc_opt=paths=source_relative",
	}, fileArgs...)
	if err := sh.Run("protoc", args...); err != nil {
		return err
	}

	return nil
}
