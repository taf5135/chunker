package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

const (
	splitPath = "./data/split/"
	mergePath = "./data/merge"
	smallTxt  = "small.txt"
	bigBin    = "big.bin"
)

var (
	splitSmallPath  = filepath.Join(splitPath, smallTxt)
	splitBigBinPath = filepath.Join(splitPath, bigBin)
	mergeSmallPath  = filepath.Join(mergePath, smallTxt)
	mergeBigBinPath = filepath.Join(mergePath, bigBin)
)

func TestGenerateRandomDataFile(t *testing.T) {
	//not really a test, just a repeatable way to generate a random file
	pcg := rand.NewPCG(0x7294398fd3289e8d, 0xFEEDEEBEE2223344)

	holder := make([]byte, 8)
	f, err := os.Create(splitBigBinPath)
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
	err := splitFileByPath(splitSmallPath, mergePath, 10485760)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSplitLargeFile(t *testing.T) {
	err := splitFileByPath(splitBigBinPath, mergePath, 10485760)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMergeSmallFile(t *testing.T) {
	err := assembleFileByPath(mergePath, regexp.MustCompile("small\\.txt"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestMergeLargeFile(t *testing.T) {
	err := assembleFileByPath(mergePath, regexp.MustCompile("big\\.bin"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestSplitSmallFileGUI(t *testing.T) { //BUG: remove data\split\small.txt.gz: The process cannot access the file because it is being used by another process.
	f, err := os.Open(splitSmallPath)
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
	f, err := os.Open(splitBigBinPath)
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
	f, err := os.Open(mergeSmallPath + ".gz_cpart0")
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
	var files []io.ReadCloser

	for i := range 8 {
		f, err := os.Open(mergeBigBinPath + fmt.Sprintf(".gz_cpart%d", i))
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

	preSplitHash, err := computeHash(splitBigBinPath)
	if err != nil {
		t.Fatal(err)
	}

	err = splitFileByPath(splitBigBinPath, mergePath, 10485760)
	if err != nil {
		t.Fatal(err)
	}

	err = assembleFileByPath(mergePath, regexp.MustCompile("big\\.bin"))
	if err != nil {
		t.Fatal(err)
	}

	postSplitHash, err := computeHash(mergeBigBinPath)
	if err != nil {
		t.Fatal(err)
	}

	if fmt.Sprintf("%x", preSplitHash) != fmt.Sprintf("%x", postSplitHash) {
		t.Fatal("preSplitHash != postSplitHash")
	}
}

func computeHash(path string) ([]byte, error) {
	hasher := sha256.New()

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	if _, err = io.Copy(hasher, f); err != nil {
		return nil, err
	}

	return hasher.Sum(nil), nil
}
