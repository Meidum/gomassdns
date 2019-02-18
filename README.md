# gomassdns, a Go wrapper for Massdns

## Install

All you need is just install [massdns](https://github.com/blechschmidt/massdns) on your machine.

We use https://github.com/miekg/dns for parsing and output.

## Usage Example:

```Go
func exmpl() {

	// gen MassDns struct from massdns binary path
	md, err := gomassdns.New("/usr/bin/massdns")
	if err != nil {
		log.Fatal(err)
	}

	// or massdns command
	md, err := gomassdns.New("massdns")
	if err != nil {
		log.Fatal(err)
	}
	
	// set resolvers from slice
	resolvers := []string{"8.8.8.8", "1.1.1.1"}
	if err := md.SetResolversSlice(resolvers); err != nil {
		log.Fatal(err)
	}

	// or from file
	if err := md.SetResolversFile("./resolvers.txt"); err != nil {
		log.Fatal(err)
	}

	// get output chan for dns.RR
	oc := md.GetOutput()

	// run massdns with input from chan
	domains := make(chan string)
	if err := md.DoFromChan("SOA", domains); err != nil {
		log.Fatal(err)
	}
	
	// or run massdns with input from file
	if err := md.DoFromFile("SOA", "./domains.txt"); err != nil {
		log.Fatal(err)
	}

	// remove all tmp/side files
	md.Clean()
}

```
