# gomassdns, a Go wrapper for Massdns

## Install

You'll need to install [massdns](https://github.com/blechschmidt/massdns) on your machine.

After that, you can use this library.

Example:

```Go
func exmpl() {

    var md MassDns
    
    // set filepath for massdns binary
	md.BinaryPath = "/usr/bin/massdns"

    // set output channel for dns.RR
    rc := make(chan dns.RR)
	md.Output = rc

	// set resolvers from slice
    resolvers := []string{"8.8.8.8", "1.1.1.1"}
	if err := md.SetResolvers(resolvers); err != nil {
		log.Fatal(err)
	}
    // or set resolver from file
    md.UserResolverPath = "./resolvers.txt"
    
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



