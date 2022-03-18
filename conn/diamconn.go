package conn

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"
	d "tomi/diam/diam"
	l "tomi/diam/mlog"
)

const (
	tcp_up = iota
	tcp_down
	cea_recieved
)

type Event struct {
	Eid  uint8
	Data interface{}
}

func NewEvent(eid uint8, data interface{}) Event {
	return Event{Eid: eid, Data: data}
}

type DiamConn struct {
	name              string
	diam_conf         map[string]string
	tcp_conf          map[string]string
	tcp_conn          ConnParam
	mgmt_tcp_ch       chan Event
	rcvd_tcp_ch       chan []byte
	write_tcp_ch      chan []byte
	send_mess_ch      chan d.Message
	rcv_mess_ch       chan d.Message
	mgmt_diam_conn    chan Event
	send_mess_byte_ch chan []byte
	hop_by_hop        uint32
	end_to_end        uint32
	start_time        string
	run_ind           uint32
}

var mtx sync.RWMutex

func NewDiamConn(c_send_mess_ch chan d.Message, c_rcv_mess_ch chan d.Message, c_mgmt_diam_conn chan Event, conf map[string]interface{}) DiamConn {
	conn_par := make(map[string]interface{})
	conn_par["name"] = conf["name"]
	conn_par["peer"] = conf["tcp_conf"].(map[string]string)["peer"]

	c_mgmt := make(chan Event)
	c_rcvd := make(chan []byte, 100)
	c_write := make(chan []byte, 100)

	tcp_conn := CreateNewConn(conn_par, c_mgmt, c_rcvd, c_write)

	c_diam := DiamConn{
		name:           conf["name"].(string),
		diam_conf:      conf["diam_conf"].(map[string]string),
		tcp_conf:       conf["tcp_conf"].(map[string]string),
		mgmt_tcp_ch:    c_mgmt,
		rcvd_tcp_ch:    c_rcvd,
		write_tcp_ch:   c_write,
		send_mess_ch:   c_send_mess_ch,
		rcv_mess_ch:    c_rcv_mess_ch,
		mgmt_diam_conn: c_mgmt_diam_conn,
		tcp_conn:       tcp_conn,
	}
	c_diam.init()
	return c_diam
}

func (c *DiamConn) Gen_Session_Id() string {
	var c_run uint32
	mtx.Lock()
	c.run_ind = c.run_ind + 1
	c_run = c.run_ind

	mtx.Unlock()
	ret := c.diam_conf["origin_host"] + ";" + c.start_time + ";" + strconv.FormatInt(int64(c_run), 10)

	return ret
}

func (c *DiamConn) init() {
	rand.Seed(time.Now().UnixNano())
	c.hop_by_hop = rand.Uint32()
	c.end_to_end = rand.Uint32()
	c.start_time = fmt.Sprintf("%d", Abs(time.Now().Unix()-int64(rand.Uint32()>>3)))
	c.run_ind = 0 //rand.Uint32()
	//l.Trace.Println(c.name,"RAND",c.hop_by_hop,c.end_to_end)
}

func (c *DiamConn) next_h_by_h() uint32 {
	c.hop_by_hop = c.hop_by_hop + 1
	l.Trace.Println(c.name, "NEXT", c.hop_by_hop)
	return c.hop_by_hop
}

func (c *DiamConn) next_e_to_e() uint32 {
	c.end_to_end = c.end_to_end + 1
	l.Trace.Println(c.name, "NEXT", c.end_to_end)
	return c.end_to_end
}

func (c *DiamConn) Start() {
	l.Trace.Println(c.name, "initiating diam conection:", c.tcp_conf["peer"])

	go c.tcp_conn.Start()
	for i := 0; i < 10; i++ {

		go func(cli int) {
			for {
				select {
				case mess := <-c.rcvd_tcp_ch:
					//l.Trace.Printf("%s rcvd: % x", c.name, mess)
					c_rcv := d.DecodeHeader(mess)
					//fmt.Println("GetMessageLength:",c_rcv.GetMessageLength())
					c_whole_len := uint32(len(mess))
					c_prot_len := c_rcv.GetMessageLength()
					//fmt.Println("GetMessageLength:", c_prot_len, c_whole_len)
					if c_prot_len != c_whole_len {
						l.Error.Println(c.name, "INVALID message")
						mess = mess[0:c_prot_len]
					}
					if c_rcv.IsAnswer() && (c_rcv.GetCmdCode() == d.CC_CAP_EXCH) {
						l.Trace.Println(c.name, "got CEA")
						c.mgmt_diam_conn <- NewEvent(cea_recieved, mess)
						//go c.watchdog()
						continue
					}
					if c_rcv.IsRequest() && (c_rcv.GetCmdCode() == d.CC_DEVICE_WATCHDOG) {
						l.Trace.Println(c.name, "got watchdog")
						dwa := c.createDWA(&c_rcv)
						c.write_tcp_ch <- dwa.Encode()
						//c.mgmt_diam_conn <- NewEvent(diam_up, nil)
						//go c.watchdog()
						continue
					}

					c_rvc_full_decoded := d.Decode(mess)
					c.rcv_mess_ch <- c_rvc_full_decoded
					//l.Warn.Println(c.name,"Got message:")

				case msg_to_send := <-c.send_mess_ch:

					//l.Trace.Println(c.name,"sending message:",msg_to_send)
					//l.Info.Println(c.name,"cli sending message:",cli)

					c_h_by_h := msg_to_send.Get_hop_by_hop()
					if c_h_by_h == 0 {
						c_h_by_h = c.next_h_by_h()
					}

					c_e_to_e := msg_to_send.Get_end_to_end()
					if c_e_to_e == 0 {
						c_e_to_e = c.next_e_to_e()
					}

					msg_to_send.Set_hop_by_hop(c_h_by_h)
					msg_to_send.Set_end_to_end(c_e_to_e)

					c.write_tcp_ch <- msg_to_send.Encode()
				case mgmt_event := <-c.mgmt_tcp_ch:
					l.Trace.Println(c.name, "got event:", mgmt_event)
					if mgmt_event.Eid == tcp_up {
						//TCP connection established, send CER
						cer := c.createCER()
						//l.Trace.Println(c.name,"is req:",cer.IsRequest())
						c.write_tcp_ch <- cer.Encode()
					}
				}
			}
		}(i)
	}
}

func Abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func (c *DiamConn) watchdog() {
	watch_ticker := time.NewTicker(5000 * time.Millisecond)
	for {
		select {
		case c_time := <-watch_ticker.C:
			//l.Info.Println("Tick")
			l.Trace.Println("watchdog:", c_time)
			dwr := c.createDWR()
			c.write_tcp_ch <- dwr.Encode()

		}
	}

}
func (c *DiamConn) GetOurHostAndRealm() []d.AVP {
	return []d.AVP{
		d.AVP_UTF8String(d.AVP_CODE_Origin_Host, c.diam_conf["origin_host"], d.MAND, 0),
		d.AVP_UTF8String(d.AVP_CODE_Origin_Realm, c.diam_conf["origin_realm"], d.MAND, 0),
	}
}

func (c *DiamConn) createCER() d.Message {
	cer_avp := []d.AVP{
		d.AVP_UTF8String(d.AVP_CODE_Origin_Host, c.diam_conf["origin_host"], d.MAND, 0),
		d.AVP_UTF8String(d.AVP_CODE_Origin_Realm, c.diam_conf["origin_realm"], d.MAND, 0),
		d.AVP_Address(d.AVP_CODE_Host_IP_Address, d.NewAddress(d.ENUM_ADDR_FAMILY, d.IPv4ToByte(c.diam_conf["host_ip"])), d.MAND, 0),
		d.AVP_UTF8String(d.AVP_CODE_Product_Name, "golang cli", d.MAND, 0),
		d.AVP_Unsigned32(d.AVP_CODE_Vendor_Id, 666, d.MAND, 0),
		d.AVP_Unsigned32(d.AVP_CODE_Auth_Application_Id, d.APPID_CC, d.MAND, 0),
	}

	cer := d.GenMess(d.CC_CAP_EXCH, true, false, d.APPID_COMMON, c.next_h_by_h(), c.next_e_to_e(), cer_avp)
	return cer
}

func (c *DiamConn) createDWR() d.Message {
	dwh_avp := []d.AVP{
		d.AVP_UTF8String(d.AVP_CODE_Origin_Host, c.diam_conf["origin_host"], d.MAND, 0),
		d.AVP_UTF8String(d.AVP_CODE_Origin_Realm, c.diam_conf["origin_realm"], d.MAND, 0),
	}
	cer := d.GenMess(d.CC_DEVICE_WATCHDOG, true, false, d.APPID_COMMON, c.next_h_by_h(), c.next_e_to_e(), dwh_avp)
	return cer
}
func (c *DiamConn) createDWA(dwr *d.Message) d.Message {
	dwh_avp := []d.AVP{
		d.AVP_UTF8String(d.AVP_CODE_Origin_Host, c.diam_conf["origin_host"], d.MAND, 0),
		d.AVP_UTF8String(d.AVP_CODE_Origin_Realm, c.diam_conf["origin_realm"], d.MAND, 0),
		d.AVP_Enumerated(d.AVP_CODE_Result_Code, d.LIMITED_SUCCESS, d.MAND, 0),
	}
	cer := d.GenMess(d.CC_DEVICE_WATCHDOG, true, false, d.APPID_COMMON, dwr.Get_hop_by_hop(), dwr.Get_end_to_end(), dwh_avp)
	return cer
}
