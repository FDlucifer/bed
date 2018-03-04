package cmdline

import (
	"strings"
	"unicode"

	"github.com/itchyny/bed/core"
)

var commands = []command{
	{"exi[t]", core.EventQuit},
	{"qa[ll]", core.EventQuit},
	{"q[uit]", core.EventQuit},
	{"x[it]", core.EventQuit},
	{"xa[ll]", core.EventQuit},
}

func parse(cmdline []rune) (command, error) {
	i, l := 0, len(cmdline)
	for i < l && (unicode.IsSpace(cmdline[i]) || cmdline[i] == ':') {
		i++
	}
	xs := strings.Fields(string(cmdline[i:]))
	if len(xs) == 0 {
		return command{}, nil
	}
	for _, cmd := range commands {
		if xs[0][0] != cmd.name[0] {
			continue
		}
		for _, c := range expand(cmd.name) {
			if xs[0] == c {
				return cmd, nil
			}
		}
	}
	return command{}, nil
}

func expand(name string) []string {
	var prefix, abbr string
	if i := strings.IndexRune(name, '['); i > 0 {
		prefix = name[:i]
		abbr = name[i+1 : len(name)-1]
	}
	if len(abbr) == 0 {
		return []string{name}
	}
	cmds := make([]string, len(abbr)+1)
	for i := 0; i <= len(abbr); i++ {
		cmds[i] = prefix + abbr[:i]
	}
	return cmds
}