package main

import (
	"encoding/json"
	"fmt"

	"github.com/percona/percona-toolkit/src/go/pt-galera-log-explainer/regex"
	"github.com/pkg/errors"
)

type regexList struct {
}

func (l *regexList) Help() string {
	return "List available regexes. Can be used to exclude them later"
}

func (l *regexList) Run() error {

	allregexes := regex.AllRegexes()
	allregexes.Merge(regex.PXCOperatorMap)

	out, err := json.Marshal(&allregexes)
	if err != nil {
		return errors.Wrap(err, "could not marshal regexes")
	}
	fmt.Println(string(out))
	return nil
}
