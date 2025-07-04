package main

import (
	"os"
)

func setupTestData() {
	_ = os.Mkdir("testdata", 0755)

	// Project 1
	_ = os.MkdirAll("testdata/project1/src", 0755)
	_ = os.MkdirAll("testdata/project1/dist", 0755)
	_ = os.MkdirAll("testdata/project1/secrets", 0755)
	_ = os.WriteFile("testdata/project1/main.go", []byte("package main"), 0644)
	_ = os.WriteFile("testdata/project1/src/helper.go", []byte("package src"), 0644)
	_ = os.WriteFile("testdata/project1/README.md", []byte("# Project 1"), 0644)
	_ = os.WriteFile("testdata/project1/dist/app", []byte("some app"), 0644)
	_ = os.WriteFile(
		"testdata/project1/data.bin",
		[]byte{'b', 'i', 'n', 'a', 'r', 'y', 0x00, 'c', 'o', 'n', 't', 'e', 'n', 't'},
		0644,
	)
	_ = os.WriteFile("testdata/project1/secrets/key.txt", []byte("secret key"), 0644)
	gitignore := `
# build artifacts
dist/
*.bin

# secrets
/secrets/
`
	_ = os.WriteFile("testdata/project1/.gitignore", []byte(gitignore), 0644)

	// Custom ignore file
	customIgnore := `
# ignore all helpers
**/helper.go
`
	_ = os.WriteFile("testdata/custom.ignore", []byte(customIgnore), 0644)

	// Default template file
	defaultTemplate := "Filename: {basename}\n---\n{content}\n"
	_ = os.WriteFile("testdata/.contextgrep.template.txt", []byte(defaultTemplate), 0644)
}

func cleanupTestData() {
	_ = os.RemoveAll("testdata")
}
