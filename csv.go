package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

type selector interface {
	choose(cols []string) []string
}

func parseSelector(str string) (selector, bool) {
	tokens := strings.Split(str, "-")

	if len(tokens) < 1 || len(tokens) > 2 {
		return nil, false
	}

	if len(tokens) == 1 {
		col, err := strconv.Atoi(tokens[0])
		if err != nil {
			return nil, false
		}

		return singleColumn{column: col - 1}, true
	}

	if len(tokens) == 2 {
		start, err := strconv.Atoi(tokens[0])
		if err != nil {
			return nil, false
		}

		if len(tokens[1]) == 0 {
			return fromColumn{column: start - 1}, true
		}

		stop, err := strconv.Atoi(tokens[1])
		if err != nil {
			return nil, false
		}

		return columnRange{start: start - 1, stop: stop - 1}, true
	}

	return nil, false
}

type singleColumn struct {
	column int
}

func (c singleColumn) choose(cols []string) []string {
	if c.column < len(cols) {
		return []string{cols[c.column]}
	}
	return []string{}
}

type fromColumn struct {
	column int
}

func (c fromColumn) choose(cols []string) []string {

	if c.column < len(cols) {
		return cols[c.column:]
	}

	return []string{}
}

type columnRange struct {
	start, stop int
}

func (c columnRange) choose(cols []string) []string {

	start, stop := c.start, c.stop
	flip := start > stop

	if flip {
		stop, start = start, stop
	}

	if start < 0 {
		start = 0
	}

	if stop >= len(cols) {
		stop = len(cols) - 1
	}

	result := cols[start : stop+1]

	if flip {
		top := len(result) - 1
		for i := 0; i < len(result)/2; i++ {
			result[i], result[top-i] = result[top-i], result[i]
		}
	}

	return result
}

type options struct {
	selectors []selector
	printToc  bool
	squash    bool
	tsv       bool
	raw       bool
}

func initOptions() options {
	splice := flag.String("c", "", "Comma separated list of columns to include in result")
	printToc := flag.Bool("header", false, "Dump the header row w/ index values")
	squash := flag.Bool("trim", false, "Trim rows that have no data to output")
	tsv := flag.Bool("tsv", false, "Output in tsv format")
	raw := flag.Bool("raw", false, "Output raw data")

	flag.Parse()

	// splice := flag.Arg(0)

	opts := options{}

	if len(*splice) > 0 {
		for _, arg := range strings.Split(*splice, ",") {
			if sel, ok := parseSelector(arg); ok {
				opts.selectors = append(opts.selectors, sel)
				continue
			}

			log.Fatalf("Bad selector: %s\n", arg)
		}
	}

	opts.printToc = *printToc
	opts.squash = *squash
	opts.tsv = *tsv
	opts.raw = *raw

	if len(opts.selectors) == 0 {
		opts.selectors = append(opts.selectors, fromColumn{0})
	}

	return opts
}

func dumpRows(r csv.Reader, write func([]string), selectors []selector, squash bool) {
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		outRow := []string{}
		for _, sel := range selectors {
			outRow = append(outRow, sel.choose(row)...)
		}

		// When `squash` is true, we want to omit any rows that have no data.

		for _, v := range outRow {

			if len(v) == 0 && squash {
				continue
			}

			write(outRow)
			break
		}
	}
}

func printToc(r csv.Reader) {
	header, err := r.Read()
	if err != nil {
		log.Fatal(err)
	}

	for i, h := range header {
		fmt.Printf("%4d %s\n", i+1, h)
	}
}

func main() {
	opts := initOptions()

	var outfn func([]string)

	if opts.raw {
		writer := bufio.NewWriter(os.Stdout)
		outfn = func(cols []string) {
			for _, v := range cols {
				_, err := writer.WriteString(v)
				if err != nil {
					log.Fatal(err)
				}
			}
			_, err := writer.WriteString("\n")
			if err != nil {
				log.Fatal(err)
			}
		}
	} else {
		writer := csv.NewWriter(os.Stdout)
		if opts.tsv {
			writer.Comma = '\t'
		}
		outfn = func(cols []string) {
			err := writer.Write(cols)
			if err != nil {
				log.Fatal(nil)
			}
		}
	}

	reader := csv.NewReader(os.Stdin)

	if opts.printToc {
		printToc(*reader)
	} else {
		dumpRows(*reader, outfn, opts.selectors, opts.squash)
	}
}
