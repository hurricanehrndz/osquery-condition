/*
osquery-condition appends MunkiConditions from osquery data
*/
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"howett.net/plist"
	osquery "github.com/kolide/osquery-go"
	"github.com/pkg/errors"
)
const conditionalItemsFile = "/Library/Managed Installs/ConditionalItems.plist"
var version = "dev"

func printSlice(s []string) {
	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
}

func main() {
	var (
		flQueries    = flag.String("queries", "", "path to line delimited query file")
		flSocketPath = flag.String("socket", "/var/osquery/osquery.em", "path to osqueryd socket")
	)
	flag.Parse()

	if *flQueries == "" {
		fmt.Println("No query file specified.")
		flag.Usage()
		os.Exit(1)
	}

	var conditions MunkiConditions
	if err := conditions.Load(); err != nil {
		if os.IsNotExist(errors.Cause(err)) {
			conditions = make(MunkiConditions)
		} else {
			log.Fatal(err)
		}
	}

	client, err := osquery.NewClient(*flSocketPath, 10*time.Second)
	if err != nil {
		fmt.Println("Error creating Thrift client: " + err.Error())
		os.Exit(1)
	}
	defer client.Close()

	// load queries from list
	queries := readQueries(*flQueries)

	// create a (w)rapped client using our OsqueryClient type.
	wclient := &OsqueryClient{client}

	resp, err := wclient.RunQueries(queries...)
	if err != nil {
		log.Fatal(err)
	}

	// range over the response channel and format all the responses as
	// conditions.
	res := map[string][]string{}
	for r := range resp {
		for k, v := range r {
			res[k] = append(res[k], v)
		}
	}

	for k, v := range res {
		conditions[fmt.Sprintf("osquery_%s", k)] = v
	}

	if err := conditions.Save(); err != nil {
		log.Fatal(err)
	}
}

// read queries from file
func readQueries(path string) []string {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	lr := bufio.NewReader(bytes.NewReader(data))
	var lines []string
	for {
		line, _, err := lr.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		lines = append(lines, string(line))
	}
	return lines
}

// OsqueryClient wraps the extension client.
type OsqueryClient struct {
	*osquery.ExtensionManagerClient
}

// RunQueries takes one or more SQL queries and returns a channel with all the responses.
func (c *OsqueryClient) RunQueries(queries ...string) (<-chan map[string]string, error) {
	responses := make(chan map[string]string)

	// schedule the queries in a separate goroutine
	// it doesn't wait for the responses to return.
	go func() {
		for _, q := range queries {
			resp, err := c.Query(q)
			if err != nil {
				log.Println(err)
				return
			}
			if resp.Status.Code != 0 {
				log.Printf("got status %d\n", resp.Status.Code)
				return
			}
			for _, r := range resp.Response {
				responses <- r
			}
		}
		// close the response channel when all queries finish running.
		close(responses)
	}()
	return responses, nil
}

// MunkiConditions is a data structure to hold confitional information.
type MunkiConditions map[string]interface{}

// Load populates datastructure with existing conditional information found on disk.
func (c *MunkiConditions) Load() error {
	f, err := ioutil.ReadFile(conditionalItemsFile)
	if err != nil {
		return errors.Wrap(err, "load ConditionalItems plist")
	}

	if _, err := plist.Unmarshal(f, c); err != nil {
		return errors.Wrap(err, "decode ConditionalItems plist")
	}
	return nil
}


// Save writes to disk to conditional infomation held in variable of type MunkiConditions.
func (c *MunkiConditions) Save() error {
	serialized, err := plist.MarshalIndent(c, plist.XMLFormat, "  ")
	if err != nil {
		return errors.Wrap(err, "encode ConditionalItems plist")
	}

	err = ioutil.WriteFile(conditionalItemsFile, serialized, 0644)
	if err != nil {
		return errors.Wrap(err, "encode ConditionalItems plist")
	}
	return nil
}
