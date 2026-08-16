package main

import (
	"compress/gzip"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

/*
.cpart file design:

2-byte magic number
2-byte index value
x-4 bytes of data, where x is the length of the file
*/

const magic = 0xFE26
const headerLength = 4 //TODO perhaps header should be a struct so we can easily marshal & unmarshal binary

var cpartRegexp = regexp.MustCompile("(.*)_cpart(\\d+)$")

func createCompressedFile(filePath, outFilePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	compressedFilePath := path.Join(outFilePath, path.Base(fmt.Sprintf("%s%s", filePath, ".gz")))
	comp, err := os.Create(compressedFilePath)
	if err != nil {
		return "", err
	}
	defer comp.Close()

	gz := gzip.NewWriter(comp)
	defer gz.Close()

	_, err = io.Copy(gz, f)
	if err != nil {
		return "", err
	}

	//if we don't call gz.Flush(), some data gets left off including the gzip footer. Can sometimes cause issues
	gz.Flush()

	return compressedFilePath, nil
}

func writeFile(filePath string, dat []byte, idx uint16) error {

	partFileName := fmt.Sprintf("%s_cpart%d", filePath, idx)

	f, err := os.Create(partFileName)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(dat)
	return err
}

func splitFileByPath(inFilePath string, outFilePath string, chunkSizeBytes int) error {
	var idx uint16

	cmpFileName, err := createCompressedFile(inFilePath, outFilePath)
	if err != nil {
		return err
	}

	header := make([]byte, 4)
	binary.BigEndian.PutUint16(header[:2], magic)

	cmpFile, err := os.Open(cmpFileName)
	if err != nil {
		return err
	}
	defer cmpFile.Close()

	readSize := chunkSizeBytes - len(header) //TODO duplicate code
	for {

		binary.BigEndian.PutUint16(header[2:], idx)
		dat := make([]byte, readSize)
		copy(dat, header)
		readLen, err := cmpFile.Read(dat[len(header):])

		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		dat = dat[:readLen+len(header)]

		err = writeFile(cmpFileName, dat, idx)
		if err != nil {
			return err
		}

		idx++

	}

	return nil
}

func filter(entries []os.DirEntry, fnameRegexp *regexp.Regexp) []os.DirEntry {
	preserved := make([]os.DirEntry, 0, len(entries))
	for _, e := range entries {

		if cpartRegexp.MatchString(e.Name()) {
			if fnameRegexp != nil && !fnameRegexp.MatchString(e.Name()) {
				continue
			}
			preserved = append(preserved, e)
		}

	}
	return preserved
}

func decompressFile(comp *os.File) error {
	_, err := comp.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}
	gz, err := gzip.NewReader(comp)
	if err != nil {
		return err
	}
	defer gz.Close()

	compName := comp.Name()
	lastInd := strings.LastIndex(compName, ".")
	if lastInd == -1 {
		return fmt.Errorf("file has no extension (%s)", compName)
	}

	decomp, err := os.Create(compName[:lastInd])
	if err != nil {
		return err
	}
	defer decomp.Close()

	_, err = io.Copy(decomp, gz)
	return err
}

func assembleFileByPath(dirPath string, fnameRegex *regexp.Regexp) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}

	entries = filter(entries, fnameRegex)

	if len(entries) == 0 {
		return fmt.Errorf("no cpart files found in directory %s", dirPath)
	}

	files := make([]io.ReadCloser, 0, len(entries))
	for _, entry := range entries {
		f, err := os.Open(filepath.Join(dirPath, entry.Name()))
		if err != nil {
			return err
		}
		files = append(files, f)
		defer f.Close()
	}

	return assembleFile(files)
}

func parseChunkSize(cstr string) (int, error) {
	cstr = strings.TrimSpace(strings.ToUpper(cstr))
	last := cstr[len(cstr)-1]

	flt, err := strconv.ParseFloat(cstr[:len(cstr)-1], 64)
	if err != nil {
		return 0, err
	}

	switch last {
	case 'K':
		flt = flt * 1024
	case 'M':
		flt = flt * 1048576
	case 'G':
		flt = flt * 1073741824
	}
	return int(flt), nil
}

func splitFile(file io.ReadCloser, chunkSize int) error {
	fpath, fname, err := extractFilePathAndName(file)
	if err != nil {
		return err
	}

	compressedFilePath := path.Join(fpath, fmt.Sprintf("%s%s", fname, ".gz"))
	comp, err := os.Create(compressedFilePath)
	if err != nil {
		return err
	}
	defer comp.Close()

	gz := gzip.NewWriter(comp)
	defer gz.Close()

	_, err = io.Copy(gz, file)
	if err != nil {
		return err
	}

	err = gz.Flush()
	if err != nil {
		return err
	}

	gz.Close() //TODO run tests. may be unnecessary

	_, err = comp.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	var idx uint16
	header := make([]byte, 4)
	binary.BigEndian.PutUint16(header[:2], magic)
	readSize := chunkSize - len(header)
	for {
		binary.BigEndian.PutUint16(header[2:], idx)
		dat := make([]byte, readSize)
		copy(dat, header)
		readLen, err := comp.Read(dat[len(header):])

		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		dat = dat[:readLen+len(header)]

		err = writeFile(compressedFilePath, dat, idx)
		if err != nil {
			return err
		}

		idx++
	}

	//TODO delete the intermediate compressed file

	return nil
}

func assembleFile(files []io.ReadCloser) error {

	if len(files) == 0 {
		return fmt.Errorf("no files to assemble")
	}

	dirPath, compressedName, err := extractFilePathAndName(files[0])

	sorted, err := sortFilesAndStripHeader(files)
	if err != nil {
		return err
	}

	compressedFile, err := os.Create(filepath.Join(dirPath, compressedName))
	if err != nil {
		return err
	}
	defer compressedFile.Close()
	for _, file := range sorted {
		_, err = io.Copy(compressedFile, file)
		if err != nil {
			return err
		}
	}

	//TODO delete the intermediate compressed file

	err = decompressFile(compressedFile)
	return err
}

/*
Extracts a file path and a clean "base" filename from a ReadCloser.
*/
func extractFilePathAndName(rc io.ReadCloser) (string, string, error) {
	file, ok := rc.(*os.File)
	if !ok {
		return "", "", fmt.Errorf("ReadCloser is not a os.File")
	}

	dir, name := filepath.Split(file.Name())
	name = strings.Split(name, "_cpart")[0]

	return dir, name, nil
}

func sortFilesAndStripHeader(files []io.ReadCloser) ([]io.ReadCloser, error) {

	sorted := make([]io.ReadCloser, len(files))
	headerBuf := make([]byte, headerLength)

	for fileIdx, f := range files {
		read, err := f.Read(headerBuf)
		if err != nil {
			return nil, err
		}

		if read != headerLength {
			return nil, fmt.Errorf("file %d too short", fileIdx)
		}

		if binary.BigEndian.Uint16(headerBuf[0:2]) != magic {
			return nil, fmt.Errorf("file %d does not have magic number", fileIdx)
		}

		idx := binary.BigEndian.Uint16(headerBuf[2:4])
		if idx < 0 || int(idx) >= len(files) {
			return nil, fmt.Errorf("file %d had index out of bounds: %d", fileIdx, idx)
		}

		if sorted[idx] != nil {
			return nil, fmt.Errorf("file %d had duplicate index %d", fileIdx, idx)
		}

		sorted[idx] = f
	}

	return sorted, nil
}

//TODO maybe make a contribution to gioui.org/x/explorer that adds support for the windows folder picker UI

func main() {

	if len(os.Args) <= 1 {
		GUIRun()
		return
	}

	splitCmd := flag.NewFlagSet("split", flag.ExitOnError)

	inPathPtrSplit := splitCmd.String("in", "./", "path to input file")
	outPathPtrSplit := splitCmd.String("out", "./", "path to the output directory. This dir will fill with cpart files")
	chunkSize := splitCmd.String("chunk", "10M", "chunk size in bytes. Accepts scientific notation (10K, 10M, 1G, etc.)")

	mergeCmd := flag.NewFlagSet("merge", flag.ExitOnError)
	inPathPtrMerge := mergeCmd.String("in", "./", "path to directory containing cpart files")
	fnameRegexStringPtr := mergeCmd.String("regex", ".*", "regular expression to match files against, in case there are multiple varieties of cpart file in the input dir")

	var err error
	switch os.Args[1] {
	case "split", "s":
		splitCmd.Parse(os.Args[2:])
		chunkSizeInt, err := parseChunkSize(*chunkSize)
		if err != nil {
			break
		}
		err = splitFileByPath(*inPathPtrSplit, *outPathPtrSplit, chunkSizeInt)
	case "merge", "assemble", "join", "m":
		mergeCmd.Parse(os.Args[2:])

		var fnameRegex *regexp.Regexp

		fnameRegex, err = regexp.Compile(*fnameRegexStringPtr)
		if err != nil {
			break
		}

		err = assembleFileByPath(*inPathPtrMerge, fnameRegex)
	default:
		fmt.Println("expected 'split' or 'merge' subcommand")
		os.Exit(1)
	}
	if err != nil {
		fmt.Println("Success")
	} else {
		fmt.Printf("error: %s\n", err)
	}

}
