package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/ncruces/zenity"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

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
	fn := writeCsv(path, datachan)

	fmt.Print("\n\n")
	fmt.Println("-- 完了しました --  :", time.Since(start))
	fmt.Print("\n")
	fmt.Println("-- ダウンロードフォルダにCSVファイルを保存しました --")
	fmt.Print("\n")
	fmt.Printf("        %s\n\n", fn)
	fmt.Println("-- なにかキーを押して終了してください --")
	scanner2 := bufio.NewScanner(os.Stdin)
	scanner2.Scan()
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

type runeWriter struct {
	w io.Writer
}

func (rw *runeWriter) Write(b []byte) (int, error) {
	var err error
	l := 0

	for len(b) > 0 {
		_, n := utf8.DecodeRune(b)
		if n == 0 {
			break
		}
		_, err = rw.w.Write(b[:n])
		if err != nil {
			_, err = rw.w.Write([]byte{'?'})
			if err != nil {
				break
			}
		}
		l += n
		b = b[n:]
	}
	return l, err
}



func writeCsv(path string, datachan chan [][]string) string {
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

	w := csv.NewWriter(&runeWriter{transform.NewWriter(f, japanese.ShiftJIS.NewEncoder())})
	w.UseCRLF = true

	data := <-datachan
	err = w.WriteAll(data)
	if err != nil {
		log.Fatal(err.Error())
	}
	w.Flush()
	if err := w.Error(); err != nil {
		log.Fatal(err)
	}

	return fn
}
