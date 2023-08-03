package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

var service string

const fkey = "service"
const traceidkey = "traceid"

func init() {
	flag.StringVar(&service, fkey, "", "filter which service to see")
}

func main() {
	flag.Parse()

	var buff strings.Builder

	fserv := strings.ToLower(service)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		s := scanner.Text()

		m := make(map[string]any)

		if err := json.Unmarshal([]byte(s), &m); err != nil {
			if fserv == "" {
				fmt.Println(s)
			}
			continue
		}

		if lfkey, ok := m[fkey].(string); fserv != "" && ok && strings.ToLower(lfkey) != fkey {
			continue
		}

		traceID, ok := m[traceidkey].(string)
		if !ok {
			traceID = "00000000-0000-0000-0000-000000000000"
		}

		buff.Reset()
		buff.WriteString(fmt.Sprintf("%s: %s: %s: %s: %s: %s",
			m[fkey],
			m["ts"],
			m["caller"],
			m["level"],
			traceID,
			m["msg"],
		))

		for k, v := range m {
			switch k {
			case fkey, "ts", "caller", "level", traceidkey, "msg":
				continue
			default:
				buff.WriteString(fmt.Sprintf(": %s[%v]", k, v))
			}
		}

		fmt.Println(buff.String())

	}

	if err := scanner.Err(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
