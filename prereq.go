package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	filetype "gopkg.in/h2non/filetype.v1"
)

func loadInstalledFonts() error {
	home := os.Getenv("USERPROFILE")
	elefontDir := fmt.Sprintf("%s/EleFont", strings.TrimSuffix(home, "/"))

	if !elefontDirExists(elefontDir) {
		createElefontDir(elefontDir)
	}

	files, err := ioutil.ReadDir(elefontDir)
	if err != nil {
		return err
	}
	for _, f := range files {
		fpath := fmt.Sprintf("%s/%s", elefontDir, f.Name())
		log.Printf("%s is a %t", f.Name(), validFont(fpath))
	}
	return nil
}

func elefontDirExists(dir string) bool {
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func createElefontDir(dir string) {
	err := os.Mkdir(dir, os.ModePerm)
	if err != nil {
		log.Fatalf("could not create elefont dir: %v", err)
	}
}

func validFont(fontfile string) bool {
	f, err := os.Open(fontfile)
	if err != nil {
		elog.Error(1, fmt.Sprintf("could not open file '%s' for validation: %v", fontfile, err))
		return false
	}
	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		elog.Error(1, fmt.Sprintf("could not read file '%s' for validation: %v", fontfile, err))
		return false
	}
	return filetype.IsFont(buf[:n])
}
