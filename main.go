package main

import (
	"encoding/csv"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/ncruces/zenity"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

// work
func work(fi fs.FileInfo, path string) []string {
	if !fi.IsDir() {
		return nil
	}
	p := filepath.Join(path, fi.Name())
	size, err := getSizeRecursive(p)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Printf("- %s : %d\n", fi.Name(), size)
	sl := []string{fi.Name(), strconv.FormatInt(size, 10)}
	return sl
}

// worker
func worker(_ int, ch chan fs.FileInfo, wg *sync.WaitGroup, path string, datachan chan [][]string) {
	for fi := range ch {
		sl := work(fi, path)
		if sl != nil {
			data := <-datachan
			data = append(data, sl)
			datachan <- data
		}
		wg.Done()
	}
}

func main() {
	datachan := make(chan [][]string, 1)
	data := [][]string{{"dirName", "size"}}
	datachan <- data

	path := selectFolder()
	start := time.Now()
	fmt.Println(path)

	fis := getDirFis(path)

	// store jobs in channel
	len := len(fis)
	ch := make(chan fs.FileInfo, len)
	for _, fi := range fis {
		// defer close(ch)
		ch <- fi
	}

	var wg sync.WaitGroup
	wg.Add(len)

	//make workers and start them
	c := runtime.NumCPU()
	var num int
	if len < c {
		num = len
	} else {
		num = c
	}
	for i := 0; i < num; i++ {

		go worker(i, ch, &wg, path, datachan)
	}

	wg.Wait()
	// close(datachan)
	writeCsv(path, datachan)
	fmt.Println(time.Since(start).Seconds())
}

func selectFolder() string {
	const defaultPath = ``
	path, err := zenity.SelectFile(zenity.Filename(defaultPath), zenity.Directory())
	if err != nil {
		panic(err)
	}
	return path
}

func getDirFis(path string) []fs.FileInfo {
	fis, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err.Error())
	}
	return fis
}

func getSizeRecursive(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

func writeCsv(path string, datachan chan [][]string) {
	hd, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err.Error())
	}
	fn := "du-" + filepath.Base(path) + "_" + time.Now().Format("2006-01-02-150405") + ".csv"
	fp := filepath.Join(hd, "Downloads", fn)

	f, err := os.Create(fp)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer f.Close()

	w := csv.NewWriter(transform.NewWriter(f, japanese.ShiftJIS.NewEncoder()))
	data := <-datachan
	err = w.WriteAll(data)
	if err != nil {
		panic(err)
	}
	w.Flush()
	if err := w.Error(); err != nil {
		log.Fatal(err)
	}
}
