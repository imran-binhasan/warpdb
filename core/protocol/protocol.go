package protocol

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func Parse(reader *bufio.Reader) ([]string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSpace(line)
	if len(line) == 0 || line[0] != '*' {
		return nil, fmt.Errorf("expected array got %s", line)
	}
	var count int
	_, err = fmt.Sscanf(line[1:], "%d", &count)
	if err != nil {
		return nil, err
	}

	args := make([]string, 0, count)

	for i := 0; i < count; i++ {
		eachLine, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		eachLine = strings.TrimSpace(eachLine)
		if len(eachLine) == 0 || eachLine[0] != '$' {
			return nil, fmt.Errorf("expected bulk string, got %s", eachLine)
		}
		var strLength int
		_, err = fmt.Sscanf(eachLine[1:], "%d", &strLength)
		if err != nil {
			return nil, err
		}

		data := make([]byte, strLength+2)
		_, err = io.ReadFull(reader, data)
		if err != nil {
			return nil, err
		}

		arg := string(data[:strLength])
		args = append(args, arg)

	}

	return args, nil

}

func WriteSimpleString(w io.Writer, msg string) {
	fmt.Fprintf(w, "+%s\r\n", msg)
}

func WriteError(w io.Writer, msg string) {
	fmt.Fprintf(w, "-%s\r\n", msg)
}

func WriteInteger(w io.Writer, msg int) {
	fmt.Fprintf(w, ":%d\r\n", msg)
}

func WriteBulkString(w io.Writer, msg string) {
	fmt.Fprintf(w, "$%d\r\n%s\r\n", len(msg), msg)
}

func WriteNull(w io.Writer) {
	fmt.Fprintf(w, "$-1\r\n")
}

func WriteArray(w io.Writer, items []string) {
	fmt.Fprintf(w, "*%d\r\n", len(items))
	for _, item := range items {
		WriteBulkString(w, item)
	}
}
