// supercontainer project main.go
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// TODO: rename type to a better name
type data struct {
	from     string
	to       string
	filename string
	//err      error
}

type result struct {
	data
	synced bool
	err    error
}

func main() {
	var from, to string
	var useMaxCPU bool
	var workers int
	flag.StringVar(&from, "from", "/home/johan/tmp/hackad", "The directory to sync from")
	flag.StringVar(&to, "to", "/home/johan/tmp/supercontainer", "The directory to sync to")
	flag.BoolVar(&useMaxCPU, "useMaxCPU", false, "Use max CPU")
	flag.IntVar(&workers, "workerPoolSize", 400, "The number of the worker pool size")
	flag.Parse()

	if yes, _ := exists(from); !yes {
		fmt.Println("Error, from does not exists")
		return
	}

	if useMaxCPU {
		nCPU := runtime.NumCPU()
		runtime.GOMAXPROCS(nCPU)
		fmt.Println("Number of CPUs:", nCPU)
	}

	done := make(chan struct{})
	defer close(done)

	datac, errc := listDir(done, from, to)

	c := make(chan result)
	var wg sync.WaitGroup

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			digester(done, datac, c)
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(c)
	}()

	filesSynced := 0
	for r := range c {
		if r.err != nil {
			fmt.Println("Error:", r.err)
		}
		if r.synced {
			filesSynced++
		}
	}

	fmt.Println("Files synced", filesSynced)

	if err := <-errc; err != nil {
		fmt.Println("Failed", err)
	}
}

func digester(done <-chan struct{}, datac <-chan data, c chan<- result) {
	for d := range datac {
		var err error
		synced := false
		// TODO: If the file exists, do a md5 checksum to see if the file is diffrent
		if yes, _ := exists(d.to + string(os.PathSeparator) + d.filename); !yes {
			err = createDirAndCopy(d.from, d.to, d.filename)
			if err == nil {
				synced = true
			}
		}

		select {
		case c <- result{d, synced, err}:
		case <-done:
			return
		}
	}
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

func listDir(done <-chan struct{}, fromPath, toPath string) (<-chan data, <-chan error) {
	start := len(strings.Split(fromPath, string(os.PathSeparator)))

	datac := make(chan data)
	errc := make(chan error, 1)

	go func() {

		defer close(datac)

		errc <- filepath.Walk(fromPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() && info.Mode().IsRegular() {
				arr := strings.Split(path, string(os.PathSeparator))

				sep := string(os.PathSeparator)
				if len(arr[start:len(arr)-1]) == 0 {
					sep = ""
				}

				d := data{}

				d.from = strings.Join(arr[:len(arr)-1], string(os.PathSeparator))
				d.to = toPath + sep + strings.Join(arr[start:len(arr)-1], string(os.PathSeparator)) +
					string(os.PathSeparator) + info.Name()
				d.filename = info.Name()

				select {
				case datac <- d:
				case <-done:
					return errors.New("walk canceled")
				}
			}
			return nil
		})
	}()
	return datac, errc
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
