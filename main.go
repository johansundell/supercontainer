// supercontainer project main.go
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var from, to string
	flag.StringVar(&from, "from", "/home/johan/tmp/hackad", "The directory to sync from")
	flag.StringVar(&to, "to", "/home/johan/tmp/supercontainer", "The directory to sync to")
	flag.Parse()

	listDir(from, to)
}

func createDirAndCopy(from, to, filename string) error {
	if yes, _ := exists(to); !yes {
		// Create dir
		err := os.MkdirAll(to, 0755)
		if err != nil {
			return err
		}
	}
	return cp(to+string(os.PathSeparator)+filename, from+string(os.PathSeparator)+filename)
}

func listDir(fromPath, toPath string) {
	start := len(strings.Split(fromPath, string(os.PathSeparator)))
	if yes, _ := exists(fromPath); !yes {
		fmt.Println("Error, from does not exists")
		return
	}
	filepath.Walk(fromPath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && info.Mode().IsRegular() {
			arr := strings.Split(path, string(os.PathSeparator))

			sep := string(os.PathSeparator)
			if len(arr[start:len(arr)-1]) == 0 {
				sep = ""
			}

			from := strings.Join(arr[:len(arr)-1], string(os.PathSeparator))
			to := toPath + sep + strings.Join(arr[start:len(arr)-1], string(os.PathSeparator)) +
				string(os.PathSeparator) + info.Name()

			// TODO: If the file exists, do a md5 checksum to see if the file is diffrent
			if yes, _ := exists(to + string(os.PathSeparator) + info.Name()); !yes {
				if err := createDirAndCopy(from, to, info.Name()); err != nil {
					fmt.Println("Copy error", err)
				}
			}
		}
		return nil
	})
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func cp(dst, src string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	// no need to check errors on read only file, we already got everything
	// we need from the filesystem, so nothing can go wrong now.
	defer s.Close()
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}
	return d.Close()
}
