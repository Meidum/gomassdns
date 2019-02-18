package massdns

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/miekg/dns"
)

type MassDns struct {
	BinaryPath       string        // path to massnds binary
	Output           chan<- dns.RR // chan for output records
	UserResolverPath string        // path to file with dns resolvers
	tempResolverPath string        // path to file with dns resolvers, if user give slice of resolvers
}

// Setup resolvers list from slice
func (conf *MassDns) SetResolvers(resolvers []string) error {
	rf, err := ioutil.TempFile("/tmp", "resolvers")
	if err != nil {
		return err
	}
	defer rf.Close()
	for r := range resolvers {
		rf.WriteString(resolvers[r] + "\n")
	}
	conf.tempResolverPath = rf.Name()
	return nil
}

// Remove all temp/side files
func (conf *MassDns) Clean() error {
	if conf.tempResolverPath != "" {
		os.Remove(conf.tempResolverPath)
		conf.tempResolverPath = ""
	}
	return nil
}

// Convert massdns output line to dns.RR
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

// Run massdns with input from file
func (conf *MassDns) DoFromFile(rtype string, ifile string) error {
	var rf string
	if conf.UserResolverPath != "" {
		rf = conf.UserResolverPath
	}
	if conf.tempResolverPath != "" {
		rf = conf.tempResolverPath
	}
	if rf == "" {
		return errors.New("Resolvers not set")
	}

	cmd := exec.Command(
		conf.BinaryPath,
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

	go func() {
		defer wg.Done()
		for cmdScanner.Scan() {
			rr, err := converter(cmdScanner.Text())
			if err != nil {
				// log.Fatal(err)
				continue
			}
			conf.Output <- rr
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

// Run massdns with input from chan
func (conf *MassDns) DoFromChan(rtype string, input <-chan string) error {
	var rf string
	if conf.UserResolverPath != "" {
		rf = conf.UserResolverPath
	}
	if conf.tempResolverPath != "" {
		rf = conf.tempResolverPath
	}
	if rf == "" {
		return errors.New("Resolvers not set")
	}

	cmd := exec.Command(
		conf.BinaryPath,
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

	go func() {
		defer wg.Done()
		for cmdScanner.Scan() {
			rr, err := converter(cmdScanner.Text())
			if err != nil {
				log.Fatal(err)
				continue
			}
			conf.Output <- rr
		}
	}()
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		fmt.Println("Done")
		return err
	}
	fmt.Println("Done")
	wg.Wait()
	return nil
}
