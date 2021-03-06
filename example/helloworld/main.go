package main

import (
	"fmt"
	"net"

	"github.com/go-git/go-billy/v5/memfs"

	nfs "github.com/willscott/go-nfs"
	nfshelper "github.com/willscott/go-nfs/helpers"
)

func main() {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		fmt.Printf("Failed to listen: %v\n", err)
		return
	}
	fmt.Printf("Server running at %s\n", listener.Addr())

	mem := memfs.New()
	f, err := mem.Create("hello.txt")
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}
	_, _ = f.Write([]byte("hello world"))
	_ = f.Close()

	handler := nfshelper.NewNullAuthHandler(mem)
	cacheHelper := nfshelper.NewCachingHandler(handler)
	fmt.Printf("%v", nfs.Serve(listener, cacheHelper))
}
