package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"regexp"
	"testing"
)

func TestGenerateRandomDataFile(t *testing.T) {
	//not really a test, just a repeatable way to generate a random file
	pcg := rand.NewPCG(0x7294398fd3289e8d, 0xFEEDEEBEE2223344)

	holder := make([]byte, 8)
	f, err := os.Create("./data/split/big.bin")
	if err != nil {
		t.Fatal(err)
	}

	for range 10000000 {
		binary.BigEndian.PutUint64(holder, pcg.Uint64())
		_, err = f.Write(holder)
		if err != nil {
			t.Fatal(err)
			return
		}
	}

}

func TestSplitSmallFile(t *testing.T) {
	err := splitFileByPath("./data/split/small.txt", "./data/merge", 10485760)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSplitLargeFile(t *testing.T) {
	err := splitFileByPath("./data/split/big.bin", "./data/merge", 10485760)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMergeSmallFile(t *testing.T) {
	//err := assembleFile("./data/merge", "./data", regexp.MustCompile("small\\.txt"), false)
	err := assembleFileByPath("./data/merge", regexp.MustCompile("small\\.txt"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestMergeLargeFile(t *testing.T) {
	err := assembleFileByPath("./data/merge", regexp.MustCompile("big\\.bin"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestSplitSmallFileGUI(t *testing.T) {
	f, err := os.Open("./data/split/small.txt")
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	err = splitFile(f, 10485760)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSplitLargeFileGUI(t *testing.T) {
	f, err := os.Open("./data/split/big.bin")
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	err = splitFile(f, 10485760)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMergeSmallFileGUI(t *testing.T) {
	f, err := os.Open("./data/merge/small.txt.gz_cpart0")
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	err = assembleFile([]io.ReadCloser{f})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMergeLargeFileGUI(t *testing.T) {
	files := []io.ReadCloser{}

	for i := range 8 {
		f, err := os.Open(fmt.Sprintf("./data/merge/big.bin.gz_cpart%d", i))
		if err != nil {
			t.Fatal(err)
		}

		files = append(files, f)
	}

	err := assembleFile(files)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFull(t *testing.T) {
	//TODO end-to-end test of a large file being chunked, merged, and hashed with the original version to see if it matches
}
