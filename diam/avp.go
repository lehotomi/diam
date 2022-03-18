package diam

import (
	"bytes"
	bin "encoding/binary"
	"fmt"
	// l "tomi/diam/mlog"
	"strconv"
	"time"
	l "tomi/diam/mlog"
)

/*
    0                   1                   2                   3
    0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |                           AVP Code                            |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |V M P r r r r r|                  AVP Length                   |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |                        Vendor-ID (opt)                        |
   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
   |    Data ...
   +-+-+-+-+-+-+-+-+
*/

type AVP struct {
	avp_code       uint32
	vendor_flag    bool
	mandatory_flag bool
	vendor_id      uint32
	data           interface{}
	format         int
}

func Basic_AVP(code uint32, avp_format int, value interface{}, mandatory_flag bool, vendor_id uint32) AVP {
	n_avp := AVP{avp_code: code, format: avp_format, vendor_id: vendor_id, vendor_flag: false, mandatory_flag: false, data: value}

	if mandatory_flag {
		n_avp.Set_mandatory_flag(true)
	}

	if vendor_id > 0 {
		n_avp.set_vendor_flag(true)
	}

	return n_avp
}

func (a *AVP) GetVendorId() uint32 {
	return a.vendor_id
}

func (a *AVP) GetAVPCode() uint32 {
	return a.avp_code
}

func (a *AVP) set_vendor_flag(f bool) {
	a.vendor_flag = f
}

func (a *AVP) GetValue() interface{} {
	return a.data
}

func (a *AVP) IsTheSameAVP(vendor uint32, avp_code uint32) bool {
	if a.GetVendorId() == vendor && a.GetAVPCode() == avp_code {
		return true
	}
	return false
}

func (a *AVP) IsGrouped() bool {
	if a.format == Avp_Grouped {
		return true
	}
	return false
}

func (a *AVP) GetGroupAVPs() []AVP {
	if !a.IsGrouped() {
		return nil
	}
	return a.data.([]AVP)
}

func (a *AVP) FindAVP(vendor_id uint32, avp_code uint32) *AVP {
	if !a.IsGrouped() {
		return nil
	}
	avps := a.data.([]AVP)
	for i := 0; i < len(avps); i++ {
		v := &avps[i]
		//for _, v := range d.avps {
		//fmt.Println("find:")
		if (v.GetAVPCode() == avp_code) && (v.GetVendorId() == vendor_id) {
			return v
		}
	}
	return nil
}

func (a *AVP) GetStringValue() string {
	return fmt.Sprint(a.data)
}

func (a *AVP) SetIntValue(new_val int) {
	switch a.data.(type) {
	case uint32:
		a.data = uint32(new_val)

	case int32:
		a.data = int32(new_val)

	case uint64:
		a.data = uint64(new_val)

	case int64:
		a.data = int64(new_val)

	case float32:
		a.data = float32(new_val)

	case float64:
		a.data = float64(new_val)
	case string:
		a.data = fmt.Sprint(new_val)
	case time.Time:
		a.data = time.Unix(int64(new_val), 0)
	default:
		l.Warn.Printf("Unable to convert %d to type %T", new_val, a.data)

	}
}

func (a *AVP) GetIntValue() int {
	switch a.data.(type) {
	case uint32:
		return int(a.data.(uint32))

	case int32:
		return int(a.data.(int32))

	case uint64:
		return int(a.data.(uint64))

	case int64:
		return int(a.data.(int64))

	case float32:
		return int(a.data.(float32))

	case float64:
		return int(a.data.(float64))

	case string:
		intValue, err := strconv.Atoi(a.data.(string))
		if err == nil {
			return intValue
		}
	case time.Time:
		return int((a.data.(time.Time)).Unix())

	}
	l.Warn.Printf("Unable to convert %T %s to int", a.data, a.data)
	return -1 //,errors.New("cannot convert "+fmt.Sprint(a.data)+" to int")
}

func (a *AVP) Set_mandatory_flag(f bool) {
	a.mandatory_flag = f
}

func (a *AVP) GetType() string {
	return fmt.Sprintf("%T", a.data)
}

func (a *AVP) Encode() []byte {
	head_size := int32(8)
	if a.vendor_id > 0 {
		head_size = 12
	}
	avp_len := head_size

	header := make([]byte, head_size)
	bin.BigEndian.PutUint32(header[0:4], a.avp_code)

	copy(header[5:8], []byte{0xff, 0xff, 0xff})

	if a.vendor_flag {
		bin.BigEndian.PutUint32(header[8:12], a.vendor_id)
	}

	var data []byte

	switch a.format {
	case Avp_Integer32, Avp_Enumerated:
		data = int32ToByteArray(a.data.(int32))
		avp_len += 4
	case Avp_Unsigned32:
		data = uint32ToByteArray(a.data.(uint32))
		avp_len += 4
	case Avp_Unsigned64:
		data = uint64ToByteArray(a.data.(uint64))
		avp_len += 8
	case Avp_Integer64:
		data = int64ToByteArray(a.data.(int64))
		avp_len += 8
	case Avp_Float32:
		data = float32ToByteArray(a.data.(float32))
		avp_len += 4
	case Avp_Float64:
		data = float64ToByteArray(a.data.(float64))
		avp_len += 8
	case Avp_OctetString:
		data = a.data.([]byte)
		avp_len += int32(len(data))
	case Avp_Time:
		c_time := a.data.(time.Time)
		data = uint32ToByteArray(uint32(c_time.Unix()) + uint32(2208988800))
		avp_len += 4
	case Avp_UTF8String:
		data = []byte(a.data.(string))
		avp_len += int32(len(data))
	case Avp_Address:
		c_address := a.data.(Address)
		data = uint16ToByteArray(c_address.family)
		data = append(data, c_address.addr...)
		avp_len += int32(len(data))
	case Avp_Grouped:
		data = a.Encode_group()
		avp_len += int32(len(data))
	case Avp_code_unknown:
		data = a.data.([]byte)
		avp_len += int32(len(data))
	default:
		fmt.Println("unknown avp format:", a, a.format)
		panic("unknown avp format")
	}

	copy(header[4:8], int32ToByteArray(avp_len))

	if a.vendor_flag {
		header[4] |= 0b10000000
	}
	if a.mandatory_flag {
		header[4] |= 0b01000000
	}

	all := append(header, data...)

	c_size := len(all)
	c_mod := c_size % 4
	padd := [4]byte{0x0, 0x0, 0x0, 0x0}
	if c_mod != 0 {
		all = append(all, padd[0:4-c_mod]...)
	}
	return all
}

func int32ToByteArray(in int32) []byte {
	//ret := make([]byte,4)
	buf := new(bytes.Buffer)
	bin.Write(buf, bin.BigEndian, in)
	return buf.Bytes()
}

func int64ToByteArray(in int64) []byte {
	//ret := make([]byte,4)
	buf := new(bytes.Buffer)
	bin.Write(buf, bin.BigEndian, in)
	return buf.Bytes()
}

func uint32ToByteArray(in uint32) []byte {
	//ret := make([]byte,4)
	buf := new(bytes.Buffer)
	bin.Write(buf, bin.BigEndian, in)
	return buf.Bytes()
}

func uint16ToByteArray(in uint16) []byte {
	//ret := make([]byte,4)
	buf := new(bytes.Buffer)
	bin.Write(buf, bin.BigEndian, in)
	return buf.Bytes()
}

func uint64ToByteArray(in uint64) []byte {
	buf := new(bytes.Buffer)
	bin.Write(buf, bin.BigEndian, in)
	return buf.Bytes()
}

func float32ToByteArray(in float32) []byte {
	//ret := make([]byte,4)
	buf := new(bytes.Buffer)
	bin.Write(buf, bin.BigEndian, in)
	return buf.Bytes()
}

func float64ToByteArray(in float64) []byte {
	//ret := make([]byte,4)
	buf := new(bytes.Buffer)
	bin.Write(buf, bin.BigEndian, in)
	return buf.Bytes()
}

func byteArrayToUint16(in []byte) uint16 {
	return bin.BigEndian.Uint16(in)
}
func byteArrayToUint32(in []byte) uint32 {
	return bin.BigEndian.Uint32(in)
}

func byteArrayToUint64(in []byte) uint64 {
	return bin.BigEndian.Uint64(in)
}

func byteArrayToFloat32(in []byte) float32 {
	//ret := make([]byte,4)
	var out float32
	buf := bytes.NewReader(in)
	bin.Read(buf, bin.BigEndian, &out)
	return out
}

func byteArrayToFloat64(in []byte) float64 {
	var out float64
	buf := bytes.NewReader(in)
	bin.Read(buf, bin.BigEndian, &out)
	return out
}

func byteArrayToInt32(in []byte) int32 {
	var out int32
	buf := bytes.NewReader(in)
	bin.Read(buf, bin.BigEndian, &out)
	return out
}

func byteArrayToInt64(in []byte) int64 {
	var out int64
	buf := bytes.NewReader(in)
	bin.Read(buf, bin.BigEndian, &out)
	return out
}

func (a *AVP) Encode_group() []byte {
	var data []byte
	var group_data []AVP = a.data.([]AVP)

	for _, s := range group_data {
		data = append(data, s.Encode()...)
	}

	c_size := len(data)
	c_mod := c_size % 4

	if c_mod != 0 {
		panic("error in padding") //TODO error
	}
	return data
}

func Decode_AVPs(in []byte) []AVP {
	var ret []AVP

	all_len := uint32(len(in))
	avps := in
	i := 1
	for len(avps) > 0 {
		if len(avps) < 8 {
			l.Warn.Printf("avp length error, length less than 8: content % x", avps)
			return ret
		}
		c_avp_code := byteArrayToUint32(avps[0:4])

		c_length_b := make([]byte, 4)
		copy(c_length_b[0:4], avps[4:8])

		c_flags := uint8(c_length_b[0])

		c_vendor_flag := false
		if (c_flags & 0b10000000) > 0 {
			c_vendor_flag = true
		}

		c_mandatory_flag := false
		if (c_flags & 0b01000000) > 0 {
			c_mandatory_flag = true
		}

		var vendor_id uint32 = 0

		if c_vendor_flag {
			if len(avps) < 12 {
				l.Warn.Printf("avp length error, less than 12: content % x", avps)
				return ret
			}
			vendor_id = byteArrayToUint32(avps[8:12])
		}

		c_length_b[0] = 0
		c_length := byteArrayToUint32(c_length_b)

		all_len = all_len - c_length
		c_avp := avps[0:c_length]

		padded_len := c_length
		c_mod := c_length % 4
		if c_mod != 0 {
			padded_len = padded_len + (4 - c_mod)
		}
		avps = avps[padded_len:]

		i = i + 1
		c_dec_avp := Decode_AVP(c_avp_code, c_vendor_flag, c_mandatory_flag, vendor_id, c_avp)
		ret = append(ret, c_dec_avp)
	}
	return ret
}

func Decode_AVP(code uint32, vendor_flag bool, mandatory_flag bool, vendor_id uint32, all_avp_b []byte) AVP {
	var avp AVP
	var data_curr interface{}

	c_avp_format_by_code := Avp_code_unknown
	//TODO figure out format
	dict_entry := LookUpAvp(code, vendor_id)

	c_avp_format_by_code = dict_entry.avptype

	var data_part []byte
	if vendor_flag {
		data_part = all_avp_b[12:]
	} else {
		data_part = all_avp_b[8:]
	}
	switch c_avp_format_by_code {
	case Avp_Integer32, Avp_Enumerated:
		if len(data_part) != 4 {
			l.Warn.Printf("avp length mismatch: type %d avp_code %d content % x", c_avp_format_by_code, code, data_part)
		}
		data_curr = byteArrayToInt32(data_part)

	case Avp_Integer64:
		if len(data_part) != 8 {
			l.Warn.Printf("avp length mismatch: type %d avp_code %d content % x", c_avp_format_by_code, code, data_part)
		}
		data_curr = byteArrayToInt64(data_part)

	case Avp_Unsigned32:
		if len(data_part) != 4 {
			l.Warn.Printf("avp length mismatch: type %d avp_code %d content % x", c_avp_format_by_code, code, data_part)
		}
		data_curr = byteArrayToUint32(data_part)

	case Avp_Unsigned64:
		if len(data_part) != 8 {
			l.Warn.Printf("avp length mismatch: type %d avp_code %d content % x", c_avp_format_by_code, code, data_part)
		}
		data_curr = byteArrayToUint64(data_part)

	case Avp_Float32:
		if len(data_part) != 4 {
			l.Warn.Printf("avp length mismatch: type %d avp_code %d content % x", c_avp_format_by_code, code, data_part)
		}
		data_curr = byteArrayToFloat32(data_part)

	case Avp_Float64:
		if len(data_part) != 8 {
			l.Warn.Printf("avp length mismatch: type %d avp_code %d content % x", c_avp_format_by_code, code, data_part)
		}
		data_curr = byteArrayToFloat64(data_part)

	case Avp_OctetString, Avp_IPAddress:
		data_curr = data_part

	case Avp_UTF8String:
		data_curr = string(data_part)

	case Avp_Address:
		if len(data_part) != 6 {
			l.Warn.Printf("avp length mismatch: type %d avp_code %d content % x", c_avp_format_by_code, code, data_part)
		}
		c_add_fam := byteArrayToUint16(data_part[0:2])
		data_curr = Address{family: c_add_fam, addr: data_part[2:]}

	case Avp_Time:
		if len(data_part) != 4 {
			l.Warn.Printf("avp length mismatch: type %d avp_code %d content % x", c_avp_format_by_code, code, data_part)
		}
		c_unix_time := byteArrayToUint32(data_part) - uint32(2208988800)
		data_curr = time.Unix(int64(c_unix_time), 0)

	case Avp_Grouped:
		data_curr = Decode_AVPs(data_part)

	case Avp_code_unknown:
		l.Warn.Printf("unknown avp: avp_code %d content % x", code, data_part)
		data_curr = data_part

	default:
		fmt.Println("TO BE ADDED!:", dict_entry)
		data_curr = data_part
	}
	//
	avp = AVP{
		avp_code:       code,
		format:         c_avp_format_by_code,
		vendor_id:      vendor_id,
		vendor_flag:    vendor_flag,
		mandatory_flag: mandatory_flag,
		data:           data_curr,
	}
	return avp
}

func AvpStringToConst(in string) int {
	switch in {
	case "OctetString", "IPFilterRule":
		return Avp_OctetString
	case "Unsigned32", "AppId", "VendorId":
		return Avp_Unsigned32
	case "Unsigned64":
		return Avp_Unsigned64
	case "Integer32":
		return Avp_Integer32
	case "Integer64":
		return Avp_Integer64
	case "Float32":
		return Avp_Float32
	case "Float64":
		return Avp_Float64
	case "UTF8String", "DiameterURI", "DiameterIdentity":
		return Avp_UTF8String
	case "Enumerated":
		return Avp_Enumerated
	case "Grouped":
		return Avp_Grouped
	case "Time":
		return Avp_Time
	case "Address":
		return Avp_Address
	case "IPAddress":
		return Avp_IPAddress
	default:
		return -1
	}

}
