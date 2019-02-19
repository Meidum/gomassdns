package gomassdns

import (
	"bufio"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"

	"github.com/miekg/dns"
)

type MassDns struct {
	binaryPath        string      // path to massnds binary
	output            chan dns.RR // chan for output records
	userResolversPath string      // path to file with dns resolvers
	tempResolversPath string      // path to temp file with dns resolvers
}

// New - MassDns generator
func New() (*MassDns, error) {
	var md MassDns

	// check default massdns command
	c := "massdns"
	cmd := exec.Command("/bin/sh", "-c", "command -v "+c)
	if err := cmd.Run(); err == nil {
		// set command if exist
		md.binaryPath = c
	}

	// generate output channel
	oc := make(chan dns.RR)
	md.output = oc
	return &md, nil
}

// SetBinaryPath - setup binary path/command for massdns
func (md *MassDns) SetBinaryPath(bp string) error {
	// check is file/command exist
	cmd := exec.Command("/bin/sh", "-c", "command -v "+bp)
	if err := cmd.Run(); err != nil {
		return errors.New("File or command not found")
	}

	// set binary path
	md.binaryPath = bp
	return nil
}

// GetOutput - get massdns output chan for dns.RR
func (md *MassDns) GetOutput() <-chan dns.RR {
	oc := md.output
	return oc
}

// SetResolversSlice - setup resolvers from slice
func (md *MassDns) SetResolversSlice(resolvers []string) error {
	// remove old resolvers if set
	md.Clean()

	// gen random tmp file
	rf, err := ioutil.TempFile("/tmp", "resolvers")
	if err != nil {
		return err
	}
	defer rf.Close()

	// write resolvers from slice to tmp file
	for r := range resolvers {
		rf.WriteString(resolvers[r] + "\n")
	}

	// set resolvers filepath
	// temp file specify to delete after work
	md.tempResolversPath = rf.Name()
	return nil
}

// SetResolversFile - setup resolvers from user file
func (md *MassDns) SetResolversFile(rpath string) error {
	// check is file exist
	if _, err := os.Stat(rpath); os.IsNotExist(err) {
		return err
	}

	// set resolvers filepath
	// file won't be deleted after work
	md.userResolversPath = rpath
	return nil
}

// Remove all temp/side files after work
func (md *MassDns) Clean() error {
	// remove only temp resolvers file
	if md.tempResolversPath != "" {
		os.Remove(md.tempResolversPath)
		md.tempResolversPath = ""
	}
	return nil
}

// converter - convert massdns output line to dns.RR
func converter(line string) (dns.RR, error) {
	rr, err := dns.NewRR(line)
	if err != nil {
		return nil, err
	}

	if rr == nil {
		return nil, errors.New("empty record")
	}

	if n := rr.Header().Name; n == "" {
		return nil, errors.New("no domain found in the record")
	}

	return rr, nil
}

// DoFromFile - run massdns with input from file
func (md *MassDns) DoFromFile(rtype string, ifile string) error {
	var rf string
	if md.userResolversPath != "" {
		rf = md.userResolversPath
	}
	if md.tempResolversPath != "" {
		rf = md.tempResolversPath
	}
	if rf == "" {
		return errors.New("Resolvers not set")
	}

	// setup massdns with input from file
	cmd := exec.Command(
		md.binaryPath,
		"-r", rf,
		"-t", rtype,
		"-o", "S",
		ifile,
	)

	var wg sync.WaitGroup
	wg.Add(1)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	cmdScanner := bufio.NewScanner(stdout)

	// read and parse STDOUT from massdns
	go func() {
		defer wg.Done()
		for cmdScanner.Scan() {
			rr, err := converter(cmdScanner.Text())
			if err != nil {
				// log.Fatal(err)
				continue
			}
			md.output <- rr
		}
	}()

	if err := cmd.Start(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}

	wg.Wait()
	return nil
}

// DoFromChan - run massdns with input from chan
func (md *MassDns) DoFromChan(rtype string, input <-chan string) error {
	var rf string
	if md.userResolversPath != "" {
		rf = md.userResolversPath
	}
	if md.tempResolversPath != "" {
		rf = md.tempResolversPath
	}
	if rf == "" {
		return errors.New("Resolvers not set")
	}

	// setup massdns with input from STDIN
	cmd := exec.Command(
		md.binaryPath,
		"-r", rf,
		"-t", rtype,
		"-o", "S",
	)

	var wg sync.WaitGroup
	wg.Add(1)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	// write to massdns STDIN from chan
	go func() {
		defer stdin.Close()
		for d := range input {
			stdin.Write([]byte(d + "\n"))
		}
	}()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	cmdScanner := bufio.NewScanner(stdout)

	// read and parse STDOUT from massdns
	go func() {
		defer wg.Done()
		for cmdScanner.Scan() {
			rr, err := converter(cmdScanner.Text())
			if err != nil {
				// log.Fatal(err)
				continue
			}
			md.output <- rr
		}
	}()

	if err := cmd.Start(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}

	wg.Wait()
	return nil
}
