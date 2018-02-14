package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"

	"github.com/go-yaml/yaml"
)

func main() {
	// TODO: allow passing in workdir
	err := os.Chdir("example/")
	if err != nil {
		log.Fatal(err)
	}

	// Read config file
	// TODO: allow passing path to report-card.yml
	config, err := readReportCardConfig("report-card.yml")
	if err != nil {
		log.Fatal(err)
	}

	// For each check, execute the check
	results, err := runChecks(config)
	if err != nil {
		log.Fatal(err)
	}

	output := Output{
		Score:   computeScore(results),
		Results: results,
	}

	// Print results
	JSONOutput := false
	if JSONOutput {
		out, err := json.MarshalIndent(output, "", "    ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(out))
	} else {
		prettyPrintOutput(output)
	}
}

type Output struct {
	Score   float64           `json:"score"`
	Results map[string]Result `json:"results"`
}

func round(f float64) float64 {
	return math.Floor(f + .5)
}

func computeScore(results map[string]Result) float64 {
	failures := 0
	for _, r := range results {
		if r.Outcome == "failure" {
			failures++
		}
	}

	score := (1 - float64(failures)/float64(len(results))) * 100
	return round(score)
}

type ConfigCheck struct {
	Type   string                 `yaml:"Type"`
	Config map[string]interface{} `yaml:"Config"` // Will need different configuration depending on check type
}

type ReportCardConfig struct {
	Version string                 `yaml:"Version"`
	Checks  map[string]ConfigCheck `yaml:"Checks"`
}

func readReportCardConfig(path string) (ReportCardConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return ReportCardConfig{}, err
	}

	// parse checks
	conf := ReportCardConfig{}

	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		return ReportCardConfig{}, err
	}

	// validate checks (e.g. perhaps a check takes a whitelist or blacklist, but not both)
	// TODO

	return conf, nil
}

// TODO: Add outcomes enum
//- success
//- warning
//- failure

func runChecks(conf ReportCardConfig) (map[string]Result, error) {
	// TODO: parallelize
	results := map[string]Result{}
	for title, c := range conf.Checks {
		var check Check
		switch c.Type {
		case "CheckFileExists":
			// TODO: check for errors instead of type assertion, since this is user config
			// Maybe: do a bunch of validation upfront?
			check = &CheckFileExists{
				Path: c.Config["Path"].(string),
			}
		case "CheckFileMD5":
			check = &CheckFileMD5{
				Path: c.Config["Path"].(string),
				Hash: c.Config["Hash"].(string),
			}
		case "CheckFileHasString":
			check = &CheckFileHasString{
				Path:   c.Config["Path"].(string),
				String: c.Config["String"].(string),
			}
		case "CheckFileIsValidJSON":
			check = &CheckFileIsValidJSON{
				Path: c.Config["Path"].(string),
			}
		case "CheckFileHasJSONSchema":
			check = &CheckFileHasJSONSchema{
				Path:       c.Config["Path"].(string),
				SchemaPath: c.Config["Schema"].(string),
			}
		case "CheckNodeDependencies":
			blacklist := []string{}
			for _, item := range c.Config["Blacklist"].([]interface{}) {
				blacklist = append(blacklist, item.(string))
			}
			check = &CheckNodeDependencies{
				Blacklist: blacklist,
			}
		case "CheckDockerBaseImage":
			whitelist := []string{}
			for _, item := range c.Config["Whitelist"].([]interface{}) {
				whitelist = append(whitelist, item.(string))
			}
			check = &CheckDockerBaseImage{
				Whitelist: whitelist,
			}
		default:
			fmt.Printf("skipping %s...\n", title)
			continue
		}

		// Do whatever
		r, err := check.Execute()
		if err != nil {
			return nil, err
		}
		results[title] = r
	}

	return results, nil
}
