package ChanDB

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
)

//implement all of the header functions
const HeaderBytes = 128

/**
Database header storing version and records information, records only stores the number of
active records in the database, deleted records are not accounted for
*/
type Header struct {
	Records int64  `json:"records"`
	Version string `json:"version"`
}

//update header info in the database file
func (h *Header) Write(file *os.File) (retError error) {
	header, err := json.Marshal(h)
	if err != nil {
		return err
	}

	if len(header)+2 >= HeaderBytes {
		return errors.New("header size exceeding " + strconv.Itoa(HeaderBytes) + " bytes, failed to write header")
	}

	_, err = file.WriteAt([]byte(" "+string(header)+strings.Repeat("\x00", HeaderBytes-len(header)-2)+"\n"), 0)

	if err != nil {
		return err
	}

	return file.Sync()
}

//read the header info
func (h *Header) Read(file *os.File) (retError error) {
	initialPosition, err := file.Seek(0, io.SeekCurrent)
	defer file.Seek(initialPosition, io.SeekStart)

	if err != nil {
		return err
	}

	_, err = file.Seek(0, io.SeekStart)

	if err != nil {
		return err
	}

	buffer := make([]byte, HeaderBytes)
	_, err = file.ReadAt(buffer, 0)
	if err != io.EOF && err != nil {
		return err
	}

	return json.Unmarshal(bytes.Trim(buffer[1:], "\x00"), h)
}
