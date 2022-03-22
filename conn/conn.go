package conn

import (
	"bytes"
	bin "encoding/binary"
	l "github.com/lehotomi/diam/mlog"
	"net"
	"time"
)

const (
	DOWN = 0
	UP   = 1
)

type ConnParam struct {
	peer     string
	name     string
	Conn     net.Conn
	mgmt_ch  chan Event
	rcvd_ch  chan []byte
	write_ch chan []byte
	state    int
}

func CreateNewConn(conf map[string]interface{}, mgmt_ch chan Event, send_ch chan []byte, write_ch chan []byte) ConnParam {
	//TODO check
	c_conn := ConnParam{
		peer:     conf["peer"].(string),
		name:     conf["name"].(string),
		mgmt_ch:  mgmt_ch,
		rcvd_ch:  send_ch,
		write_ch: write_ch,
		state:    DOWN,
	}
	return c_conn
}

func (c *ConnParam) readLoop() {
	bufferb := make([]byte, 1000000)
	collect := make([]byte, 0, 10000000)

	for {
		nr, err := c.Conn.Read(bufferb)

		l.Trace.Println("read bytes")
		if err != nil {
			l.Info.Println(c.name, "error recieved from socket", err)
			c.state = DOWN
			break
		}

		collect = append(collect, bufferb[0:nr]...)

		i := 0
		for {
			if len(collect) == 0 {
				break
			}
			if len(collect) < 20 {
				l.Trace.Println("buffer contains less than 20:", collect)
				break
			}
			length_b := make([]byte, 4)
			copy(length_b, collect[0:4])
			c_version := collect[0]
			if c_version != 1 {
				l.Trace.Println("invalid diameter message header:", collect)
				collect = collect[0:0]
				break
			}
			length_b[0] = 0
			c_length := int(byteArrayToInt(length_b))

			if len(collect) < c_length {
				l.Trace.Println("c_length:", i, len(collect), c_length)
				l.Trace.Println("diam message not complete", collect)
				break
			}
			l.Trace.Println("message complete:", collect)
			l.Trace.Println("c_length:", i, c_length)

			c_part := collect[0:c_length]
			collect = collect[c_length:]
			c.rcvd_ch <- c_part
			i = i + 1
		}
	}
}

func (c *ConnParam) Start() {
	for i := 0; i < 10; i++ {
		go c.Writer(i) //??MORE
	}

	for {
		c.init()
		c.readLoop()
	}

}

func (c *ConnParam) Writer(cli int) {
	for {
		what_to_write := <-c.write_ch
		l.Trace.Println(c.name, "about to WRITE", what_to_write, c.Conn, c.state)
		l.Trace.Println(c.name, "client about to WRITE", cli)

		if c.state == DOWN || c.Conn == nil {
			l.Warn.Println(c.name, "trying to write, but tcp connection is down:", what_to_write)
			continue
		}

		_, err := c.Conn.Write(what_to_write)
		if err != nil {
			//c.state = DOWN
			//l.Info.Println(c.name,"about to write",what_to_write,c.Conn)
		}
	}

}

func (c *ConnParam) init() {
	c_dial := net.Dialer{
		Timeout: 2 * time.Second,
		LocalAddr: &net.TCPAddr{
			Port: 0,
		},
	}
	for {
		l.Trace.Println(c.name, "initiating connection:", c.peer)
		conn, err := c_dial.Dial("tcp", c.peer)
		if err != nil {
			l.Error.Println(c.name, err)
			time.Sleep(5 * time.Second)
			//os.Exit(1)
			continue
		}
		c.state = UP
		l.Trace.Println(c.name, "...connection established:", c.peer)
		c.Conn = conn
		c.mgmt_ch <- NewEvent(tcp_up, nil)
		break
	}

}

func byteArrayToInt(in []byte) int32 {
	var out int32
	buf := bytes.NewReader(in)
	bin.Read(buf, bin.BigEndian, &out)
	return out
}
