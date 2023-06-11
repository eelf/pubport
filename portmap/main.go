package main

import (
	"io"
	"net"
	"os"
	"sync"
)

func pipe(from, to *net.TCPConn) {
	_, _ = io.Copy(to, from)
	_ = to.CloseWrite()
}

func main() {
	addr, err := net.ResolveTCPAddr("tcp", os.Args[1])
	if err != nil {
		panic(err)
	}
	addrTo, err := net.ResolveTCPAddr("tcp", os.Args[2])
	if err != nil {
		panic(err)
	}
	src, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}
	for {
		from, err := src.AcceptTCP()
		if err != nil {
			panic(err)
		}

		go func() {
			defer from.Close()
			to, err := net.DialTCP("tcp", nil, addrTo)
			if err != nil {
				panic(err)
			}
			defer to.Close()

			var wg sync.WaitGroup
			wg.Add(2)
			go pipe(from, to)
			go pipe(to, from)
			wg.Wait()
		}()
	}
}
