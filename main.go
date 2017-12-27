package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func main() {
	// Read a config file
	config, err := readConfig("report-card.yml")
	if err != nil {
		log.Fatal(err)
	}

	// For each check, execute the check
	results, err := runChecks(config)
	if err != nil {
		log.Fatal(err)
	}

	// Print results
	out, err := json.MarshalIndent(results, "", "    ")
	if err != nil {
		log.Fatal(err)
	}

	// Compute score
	score := computeScore(results)
	fmt.Println(string(out))
	fmt.Printf("Score = %.1f%%\n", score)
}

func computeScore(results map[string]Result) float64 {
	failures := 0
	for _, r := range results {
		if r.Outcome == "failure" {
			failures++
		}
	}

	return (1 - float64(failures)/float64(len(results))) * 100
}

type ConfigCheck struct {
	Type   string
	Config interface{} // Will need different configuration depending on check type
}

type Config struct {
	Version string
	Checks  map[string]Check
}

func readConfig(path string) (Config, error) {
	// open file at path
	// TODO

	// parse checks
	// TODO

	// validate checks (e.g. perhaps a check takes a whitelist or blacklist, but not both)
	// TODO

	// For now, use a mock config
	conf := Config{
		Version: "1.2.3",
		Checks: map[string]Check{
			"Verify foo.txt exists": &CheckFileExists{
				Path: "foo.txt",
			},
			"Verify bar.txt exists": &CheckFileExists{
				Path: "bar.txt",
			},
			"Verify no blacklisted package.json deps": &CheckNodeDependencies{
				Blacklist: []string{
					"foo",
					"oauth",
					"babel-cli",
				},
			},
			"Verify uses a whitelisted Docker base image": &CheckDockerBaseImage{
				Whitelist: []string{
					"nodejs:foo",
					"golang:bar",
					"golang:baz",
					"alpine",
				},
			},
		},
	}

	return conf, nil
}

// TODO: Add outcomes enum
//- success
//- warning
//- failure

func runChecks(conf Config) (map[string]Result, error) {
	// TODO: parallelize
	results := map[string]Result{}
	for title, c := range conf.Checks {
		r, err := c.Execute()
		if err != nil {
			return nil, err
		}
		results[title] = r
	}

	return results, nil
}

type Check interface {
	Execute() (Result, error)
}

type Result struct {
	Outcome string
	Details string
}

type CheckNodeDependencies struct {
	Blacklist []string
}

func (c *CheckNodeDependencies) Execute() (Result, error) {
	data, err := ioutil.ReadFile("package.json")
	if err != nil {
		return Result{}, err
	}

	var packageJSON map[string]interface{}
	err = json.Unmarshal(data, &packageJSON)
	if err != nil {
		panic(err)
		//return Result{}, err
	}

	found := []string{}
	for _, key := range []string{"dependencies", "devDependencies"} {
		deps, ok := packageJSON[key].(map[string]interface{})
		if ok {
			for _, black := range c.Blacklist {
				if _, ok = deps[black]; ok {
					found = append(found, black)
				}
			}
		}
	}

	if len(found) > 0 {
		return Result{
			Outcome: "failure",
			Details: "found the following blacklisted packages: " + strings.Join(found, ","),
		}, nil
	}

	return Result{
		Outcome: "success",
	}, nil
}

type CheckFileExists struct {
	Path string
}

type CheckDockerBaseImage struct {
	Whitelist []string
}

func (c *CheckDockerBaseImage) Execute() (Result, error) {
	data, err := ioutil.ReadFile("Dockerfile")
	if err != nil {
		return Result{}, err
	}

	lines := strings.Split(string(data), "\n")
	if !strings.HasPrefix(lines[0], "FROM") {
		return Result{}, fmt.Errorf("unable to determine base image from Dockerfile")
	}

	image := strings.TrimSpace(strings.Trim(lines[0], "FROM"))
	found := false
	for _, white := range c.Whitelist {
		if image == white {
			found = true
			break
		}
	}

	if !found {
		return Result{
			Outcome: "failure",
			Details: "dockerfile uses base image not found in whitelist: " + image,
		}, nil
	}

	return Result{
		Outcome: "success",
	}, nil
}

func (c *CheckFileExists) Execute() (Result, error) {
	_, err := os.Stat(c.Path)
	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			return Result{
				Outcome: "failure",
			}, nil
		}
		return Result{}, err
	}

	return Result{
		Outcome: "success",
	}, nil
}
