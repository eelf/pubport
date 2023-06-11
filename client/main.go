package main

import (
	"context"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/eelf/pubport"
)

func serve(stream uint64, cl net.Conn, tcpCl pubport.PubPort_TcpClient, ch <-chan []byte, f func()) {
	defer cl.Close()
	defer f()
	buf := make([]byte, 4<<10)

	go func() {
		for {
			n, err := cl.Read(buf)
			if err != nil {
				return
			}

			err = tcpCl.Send(&pubport.Data{
				Stream: stream,
				Bytes:  buf[:n],
			})
			if err != nil {
				log.Println("send", err)
				return
			}
		}
	}()

	for b := range ch {
		for len(b) > 0 {
			n, err := cl.Write(b)
			if err != nil {
				log.Println("write", err)
				return
			}
			b = b[n:]
		}
	}
}

func main() {
	log.SetFlags(log.Lmicroseconds | log.Lshortfile)

	conn, err := grpc.Dial(os.Getenv("PUBPORT_SRV"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}

	cl := pubport.NewPubPortClient(conn)
	ctx := context.Background()

	tcpCl, err := cl.Tcp(ctx)
	if err != nil {
		panic(err)
	}

	md, err := tcpCl.Header()
	if err != nil {
		panic(err)
	}
	log.Println(md.Get("address")[0])

	m := map[uint64]chan []byte{}

	for {
		d, err := tcpCl.Recv()
		if err != nil {
			panic(err)
		}

		streamLocal := d.GetStream()

		ch, ok := m[streamLocal]
		if !ok {
			ch = make(chan []byte)
			m[streamLocal] = ch
			cl, err := net.Dial("tcp", os.Args[1])
			if err != nil {
				panic(err)
			}

			go serve(streamLocal, cl, tcpCl, ch, func() {
				delete(m, streamLocal)
			})
		}

		b := d.GetBytes()
		if b == nil {
			close(ch)
		} else {
			ch <- b
		}
	}
}
