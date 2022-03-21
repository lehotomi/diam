package diam

/*
  0                   1                   2                   3
       0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
      |    Version    |                 Message Length                |
      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
      | Command Flags |                  Command Code                 |
      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
      |                         Application-ID                        |
      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
      |                      Hop-by-Hop Identifier                    |
      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
      |                      End-to-End Identifier                    |
      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
      |  AVPs ...
      +-+-+-+-+-+-+-+-+-+-+-+-+-
https://www.iana.org/assignments/aaa-parameters/aaa-parameters.txt
*/

import (
	"fmt"
	l "github.com/lehotomi/diam/mlog"
	"strings"
)

type Header struct {
	cmd_flags      uint8
	cmd_code       uint32
	app_id         uint32
	hop_by_hop     uint32
	end_to_end     uint32
	message_length uint32
}

type Message struct {
	header Header
	avps   []AVP
}

func (d *Message) Set_hop_by_hop(hop_by_hop uint32) {
	d.header.hop_by_hop = hop_by_hop
}

func (d *Message) Set_end_to_end(end_to_end uint32) {
	d.header.end_to_end = end_to_end
}

func (d *Message) Set_request_flag(val bool) {
	if val {
		d.header.cmd_flags = d.header.cmd_flags | 0b10000000
	} else {
		d.header.cmd_flags = d.header.cmd_flags & 0b01111111
	}
}

func (d *Message) Get_hop_by_hop() uint32 {
	return d.header.hop_by_hop
}

func (d *Message) Get_end_to_end() uint32 {
	return d.header.end_to_end
}

func (d *Message) FindAVP(vendor_id uint32, avp_code uint32) *AVP {
	for i := 0; i < len(d.avps); i++ {
		v := &d.avps[i]
		if (v.GetAVPCode() == avp_code) && (v.GetVendorId() == vendor_id) {
			return v
		}
	}
	return nil
}

func (d *Message) FindAVPs(vendor_id uint32, avp_code uint32) []*AVP {
	var ret []*AVP
	for i := 0; i < len(d.avps); i++ {
		v := &d.avps[i]
		if (v.GetAVPCode() == avp_code) && (v.GetVendorId() == vendor_id) {
			ret = append(ret, v)
		}
	}
	return ret
}

func _GenReq(cmd_code uint32, app_id uint32, hop_by_hop uint32, end_to_end uint32, avps []AVP) Message {
	return GenMess(cmd_code, true, true, app_id, hop_by_hop, end_to_end, avps)
}

func GenMess(cmd_code uint32, request bool, proxiable bool, app_id uint32, hop_by_hop uint32, end_to_end uint32, avps []AVP) Message {
	var cmd_flags uint8 = 0
	if request {
		cmd_flags |= 0b10000000
	}

	if proxiable {
		cmd_flags |= 0b01000000
	}

	return Message{
		header: Header{
			cmd_flags:  cmd_flags,
			cmd_code:   cmd_code,
			app_id:     app_id,
			hop_by_hop: hop_by_hop,
			end_to_end: end_to_end},
		avps: avps}
}

func (d *Message) AddAVPs_Head(n_avp []AVP) {
	d.avps = append(n_avp, d.avps...)
}

func (d *Message) AddAVPs_Tail(n_avp []AVP) {
	d.avps = append(d.avps, n_avp...)
}

func (d *Message) Encode() []byte {
	var ret []byte

	header := make([]byte, 20)

	copy(header[4:8], uint32ToByteArray(d.header.cmd_code))
	header[4] = d.header.cmd_flags

	copy(header[8:12], uint32ToByteArray(d.header.app_id))

	copy(header[12:16], uint32ToByteArray(d.header.hop_by_hop))

	copy(header[16:20], uint32ToByteArray(d.header.end_to_end))

	ret = header

	var payload []byte
	for _, v := range d.avps {
		payload = append(payload, v.Encode()...)
	}

	length := int32(20)
	length += int32(len(payload))

	copy(header[0:4], int32ToByteArray(length))
	header[0] = 1

	ret = append(ret, payload...)
	return ret
}

func (d *Message) IsRequest() bool {
	if (d.header.cmd_flags & 0b10000000) != 0 {
		return true
	}
	return false
}

func (d *Message) IsAnswer() bool {
	if (d.header.cmd_flags & 0b10000000) != 0 {
		return false
	}
	return true
}

func (d *Message) GetCmdCode() uint32 {
	return d.header.cmd_code
}

func (d *Message) GetCmdFlags() uint8 {
	return d.header.cmd_flags
}

func (d *Message) GetMessageLength() uint32 {
	return d.header.message_length
}

func DecodeHeader(in []byte) Message {
	header := in[0:20]

	length_b := make([]byte, 4)

	copy(length_b[0:4], header[0:4])
	length_b[0] = 0

	c_length := byteArrayToUint32(length_b)

	cmd_code_b := make([]byte, 4)
	copy(cmd_code_b[0:4], header[4:8])

	c_cmd_flags := uint8(cmd_code_b[0])

	cmd_code_b[0] = 0

	c_cmd_code := byteArrayToUint32(cmd_code_b)

	c_app_id := byteArrayToUint32(header[8:12])

	c_hop_by_hop := byteArrayToUint32(header[12:16])
	c_end_to_end := byteArrayToUint32(header[16:20])

	return Message{
		header: Header{
			cmd_flags:      c_cmd_flags,
			cmd_code:       c_cmd_code,
			app_id:         c_app_id,
			hop_by_hop:     c_hop_by_hop,
			end_to_end:     c_end_to_end,
			message_length: c_length,
		},
	}
}

func Decode(in []byte) Message {
	header := in[0:20]

	length_b := make([]byte, 4)

	copy(length_b[0:4], header[0:4])
	length_b[0] = 0

	c_length := byteArrayToUint32(length_b)

	cmd_code_b := make([]byte, 4)
	copy(cmd_code_b[0:4], header[4:8])

	c_cmd_flags := uint8(cmd_code_b[0])

	cmd_code_b[0] = 0
	c_cmd_code := byteArrayToUint32(cmd_code_b)

	c_app_id := byteArrayToUint32(header[8:12])

	c_hop_by_hop := byteArrayToUint32(header[12:16])
	c_end_to_end := byteArrayToUint32(header[16:20])

	avp_data := in[20:]

	if c_length != uint32(len(avp_data))+20 {
		l.Warn.Println("lent:", len(avp_data), c_length)
	}
	//l.Trace.Printf("avps d dec:% x",avp_data)

	var avps_dec []AVP = Decode_AVPs(avp_data)

	c_mess := Message{
		header: Header{
			cmd_flags:  c_cmd_flags,
			cmd_code:   c_cmd_code,
			app_id:     c_app_id,
			hop_by_hop: c_hop_by_hop,
			end_to_end: c_end_to_end,
		},
		avps: avps_dec,
	}
	return c_mess
}

func (d *Message) ToString() string {
	var res []string

	res = append(res, fmt.Sprintf("%-11s %d", "cmd_code:", d.header.cmd_code))

	flag_str := ""
	if d.IsRequest() {
		flag_str += "Request"
	} else {
		flag_str += "Answer"
	}

	//fmt.Println(d.header.cmd_flags & 0b01000000)
	if d.header.cmd_flags&0b01000000 != 0 {
		flag_str += ",Proxiable"
	}

	res = append(res, fmt.Sprintf("%-11s 0x%x %s", "flags:", d.header.cmd_flags, flag_str))

	app_id := fmt.Sprintf("%d", d.header.app_id)
	res = append(res, fmt.Sprintf("%-11s %s", "app_id:", app_id))

	res = append(res, fmt.Sprintf("%-11s 0x%08x", "hop_by_hop:", d.header.hop_by_hop))
	res = append(res, fmt.Sprintf("%-11s 0x%08x", "end_by_end:", d.header.end_to_end))
	res = append(res, "----")
	res = append(res, avpsToString(d.avps, 0))
	return strings.Join(res, "\n")
}

const (
	const_pre = "    "
)

func avpsToString(avps []AVP, level int) string {
	var res []string
	pref := ""
	for k := 0; k < level; k++ {
		pref += const_pre
	}
	for _, v := range avps {
		//func LookUpAvp(avp_code uint32, vendor_id uint32) AVPDictEntry {
		c_avp_code := v.avp_code
		c_vendor_id := v.vendor_id
		dict_entry := LookUpAvp(c_avp_code, c_vendor_id)
		var cline string
		cflags := ""

		if c_vendor_id != 0 {
			cflags = "V"
		} else {
			cflags = "-"
		}

		if v.mandatory_flag {
			cflags += "M"
		} else {
			cflags += "-"
		}
		if c_vendor_id != 0 {
			if dict_entry.avptype == Avp_Grouped {
				cline = fmt.Sprintf("%sAVP: %s(%d) f=%s vnd=%d", pref, dict_entry.name, c_avp_code, cflags, c_vendor_id)
			} else {
				var c_val string
				if dict_entry.avptype != Avp_code_unknown {
					c_val = avpToValue(v)
				} else {
					c_val = fmt.Sprintf("0x%x", v.data)
				}
				cline = fmt.Sprintf("%sAVP: %s(%d) f=%s vnd=%d val=%s", pref, dict_entry.name, c_avp_code, cflags, c_vendor_id, c_val)
			}
		} else {
			if dict_entry.avptype == Avp_Grouped {
				cline = fmt.Sprintf("%sAVP: %s(%d) f=%s", pref, dict_entry.name, c_avp_code, cflags)
			} else {
				var c_val string
				if dict_entry.avptype != Avp_code_unknown {
					c_val = avpToValue(v)
				} else {
					c_val = fmt.Sprintf("0x%x", v.data)
				}
				cline = fmt.Sprintf("%sAVP: %s(%d) f=%s val=%s", pref, dict_entry.name, c_avp_code, cflags, c_val)
			}
		}

		if dict_entry.avptype == Avp_Grouped {
			child_avps, ok := v.data.([]AVP)
			if !ok {
				continue
			}
			child_string := avpsToString(child_avps, level+1)
			cline += "\n" + child_string
		}

		res = append(res, cline)

	}
	return strings.Join(res, "\n")
}

func avpToValue(avp AVP) string {
	return fmt.Sprintf("%v", avp.data)
}
