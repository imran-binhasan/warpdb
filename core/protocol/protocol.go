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
		_,err = io.ReadFull(reader, data)
		if err != nil {
			return nil, err
		}

		arg := string(data[:strLength])
		args = append(args, arg)

	}

	return args, nil

}
