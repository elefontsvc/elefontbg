package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	filetype "gopkg.in/h2non/filetype.v1"
)

type Font struct {
	ID   string `json:"id"`
	Path string `json:"path"`
	Name string `json:"name"`
}

var elefontDir string

func init() {
	home := os.Getenv("USERPROFILE")
	elefontDir = fmt.Sprintf("%s/EleFont", strings.TrimSuffix(home, "/"))
}

var installedFonts = make(map[string]Font)

func loadInstalledFonts() error {
	// home := os.Getenv("USERPROFILE")
	// elefontDir := fmt.Sprintf("%s/EleFont", strings.TrimSuffix(home, "/"))

	if !elefontDirExists(elefontDir) {
		createElefontDir(elefontDir)
	}

	files, err := ioutil.ReadDir(elefontDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		fpath := fmt.Sprintf("%s/%s", elefontDir, f.Name())
		if validFont(fpath) {
			b := md5.Sum([]byte(fpath))
			tmp := Font{}
			tmp.ID = string(b[:])
			tmp.Path = fpath
			tmp.Name = f.Name()
			installedFonts[tmp.ID] = tmp
			// log.Printf("%s has hash %s", tmp.Name, tmp.ID)
		}
	}
	elog.Info(1, fmt.Sprintf("elefont have %d installed fonts", len(installedFonts)))
	// log.Printf("%+v", installedFonts)
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
		elog.Error(1, fmt.Sprintf("could not create elefont dir: %v", err))
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
