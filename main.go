// Command osquery-condition runs osquery queries through a running osqueryd
// and merges the results into Munki's ConditionalItems.plist, so manifests can
// use them as conditions. It is meant to run as a Munki preflight script.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	osquery "github.com/osquery/osquery-go"
	"howett.net/plist"
)

// version is the release version, managed by `go tool versionbump`
// (see versionbump.yaml).
const version = "0.1.0"

const (
	conditionalItemsPath = "/Library/Managed Installs/ConditionalItems.plist"
	keyPrefix            = "osquery_"
	socketTimeout        = 10 * time.Second
)

// conditions mirrors ConditionalItems.plist. Values are untyped because other
// condition scripts write keys this tool must round-trip unchanged.
type conditions map[string]any

func main() {
	log.SetFlags(0)
	log.SetPrefix("osquery-condition: ")
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		log.Fatal(err)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("osquery-condition", flag.ContinueOnError)
	queriesPath := flags.String("queries", "", "path to line delimited query file")
	socketPath := flags.String("socket", "/var/osquery/osquery.em", "path to osqueryd socket")
	showVersion := flags.Bool("version", false, "print the version and exit")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		fmt.Println(version)
		return nil
	}
	if *queriesPath == "" {
		flags.Usage()
		return errors.New("no query file specified")
	}

	queries, err := readQueries(*queriesPath)
	if err != nil {
		return err
	}

	client, err := osquery.NewClient(*socketPath, socketTimeout)
	if err != nil {
		return fmt.Errorf("connect to osqueryd: %w", err)
	}
	defer client.Close()

	results, err := runQueries(client, queries)
	if err != nil {
		return err
	}

	// Load only after every query succeeded, so a failed run never rewrites
	// the plist with partial results.
	conds, err := loadConditions(conditionalItemsPath)
	if err != nil {
		return err
	}
	conds.merge(results)
	return saveConditions(conditionalItemsPath, conds)
}

// readQueries returns one query per non-blank line of the file at path.
func readQueries(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read queries: %w", err)
	}
	defer f.Close()

	var queries []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			queries = append(queries, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read queries: %w", err)
	}
	return queries, nil
}

// querier is the slice of *osquery.ExtensionManagerClient that runQueries needs.
type querier interface {
	QueryRows(sql string) ([]map[string]string, error)
}

// runQueries runs every query and groups the returned values by column name,
// across all rows of all queries.
func runQueries(q querier, queries []string) (map[string][]string, error) {
	results := map[string][]string{}
	for _, sql := range queries {
		rows, err := q.QueryRows(sql)
		if err != nil {
			return nil, fmt.Errorf("query %q: %w", sql, err)
		}
		for _, row := range rows {
			for column, value := range row {
				results[column] = append(results[column], value)
			}
		}
	}
	return results, nil
}

// merge stores each result column under an osquery_ prefixed key, replacing
// any value from a previous run and leaving every other key untouched.
func (c conditions) merge(results map[string][]string) {
	for column, values := range results {
		c[keyPrefix+column] = values
	}
}

// loadConditions reads the plist at path. A missing file yields an empty set,
// since Munki only creates it when some condition script writes to it.
func loadConditions(path string) (conditions, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return conditions{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load ConditionalItems plist: %w", err)
	}

	conds := conditions{}
	if _, err := plist.Unmarshal(data, &conds); err != nil {
		return nil, fmt.Errorf("decode ConditionalItems plist: %w", err)
	}
	return conds, nil
}

// saveConditions writes the plist via a temporary file and a rename, so an
// interrupted run cannot leave Munki a truncated file.
func saveConditions(path string, conds conditions) (err error) {
	data, err := plist.MarshalIndent(conds, plist.XMLFormat, "  ")
	if err != nil {
		return fmt.Errorf("encode ConditionalItems plist: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".ConditionalItems-*.plist")
	if err != nil {
		return fmt.Errorf("save ConditionalItems plist: %w", err)
	}
	defer func() {
		if err != nil {
			_ = os.Remove(tmp.Name())
		}
	}()

	_, err = tmp.Write(data)
	if err == nil {
		err = tmp.Chmod(0o644)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		return fmt.Errorf("save ConditionalItems plist: %w", err)
	}
	return nil
}
