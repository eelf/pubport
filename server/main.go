package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/eelf/pubport"
)

type PubPortServer struct {
	pubport.UnimplementedPubPortServer
}

func serve(stream uint64, cl net.Conn, server pubport.PubPort_TcpServer, ch chan []byte, f func()) {
	defer cl.Close()
	defer f()
	go func() {
		buf := make([]byte, 4<<10)
		for {
			n, err := cl.Read(buf)
			if err != nil {
				err = server.Send(&pubport.Data{
					Stream: stream,
				})
				close(ch)
				return
			}

			err = server.Send(&pubport.Data{
				Stream: stream,
				Bytes:  buf[:n],
			})
			if err != nil {
				close(ch)
				log.Println("send", err)
				return
			}
		}
	}()

	for b := range ch {
		for len(b) != 0 {
			n, err := cl.Write(b)
			if err != nil {
				log.Println("write", err)
				return
			}
			b = b[n:]
		}
	}
}

func (p PubPortServer) Tcp(server pubport.PubPort_TcpServer) error {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Println("listen", err)
		return err
	}
	defer ln.Close()

	err = server.SendHeader(metadata.Pairs("address", ln.Addr().String()))
	if err != nil {
		log.Println("SendHeader", err)
		return err
	}

	var stream uint64
	// todo sync.Map
	m := map[uint64]chan []byte{}

	go func() {
		defer ln.Close()
		defer func() {
			for _, ch := range m {
				close(ch)
			}
		}()
		for {
			d, err := server.Recv()
			if err != nil {
				log.Println("recv", err)
				return
			}

			ch, ok := m[d.GetStream()]
			if !ok {
				log.Println("no dst", d.GetStream())
				return
			}
			select {
			case ch <- d.GetBytes():
			default:
				log.Println("ch closed?")
				return
			}
		}
	}()

	for {
		cl, err := ln.Accept()
		if err != nil {
			return err
		}

		ch := make(chan []byte)
		m[stream] = ch

		streamLocal := stream
		go serve(streamLocal, cl, server, ch, func() {
			delete(m, streamLocal)
		})
		stream++
	}
}

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	srv := grpc.NewServer()

	ppServer := &PubPortServer{}
	pubport.RegisterPubPortServer(srv, ppServer)

	ln, err := net.Listen("tcp", os.Args[1])
	if err != nil {
		panic(err)
	}

	err = srv.Serve(ln)
	if err != nil {
		panic(err)
	}
}
