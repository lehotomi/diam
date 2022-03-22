package diam

import (
	"encoding/json"
	"fmt"
	l "github.com/lehotomi/diam/mlog"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

const (
	const_dict_dir = "dict"
)

type AVPDictEntry struct {
	code      uint32
	name      string
	vendor_id uint32
	avptype   int
}

var dict map[string]AVPDictEntry = make(map[string]AVPDictEntry)

var avp_enums map[string]map[int32]string = make(map[string]map[int32]string)

func Init(dir string) {
	var files []string
	var file_cont [][]byte
	fileInfo, err := ioutil.ReadDir(dir) //TODO
	if err != nil {
		fmt.Println("cannot find dict dir:", dir)
		os.Exit(1)
		return
	}

	for _, file := range fileInfo {
		c_file := file.Name()
		if strings.HasSuffix(c_file, ".json") {
			files = append(files, c_file)
		}
	}
	for _, c_file := range files {

		c_jsonfile, err := os.Open(dir + "/" + c_file)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer c_jsonfile.Close()
		byte_json, _ := ioutil.ReadAll(c_jsonfile)
		file_cont = append(file_cont, byte_json)

	}

	for c_index, c_file_cont := range file_cont {
		var result map[string]interface{}
		json.Unmarshal(c_file_cont, &result)

		avps := result["avps"].([]interface{})
		for _, v := range avps {
			v_map := v.(map[string]interface{})
			c_vendor_id := v_map["vendor-id"].(float64)
			c_code := v_map["code"].(float64)
			c_key := fmt.Sprint(c_vendor_id) + "." + fmt.Sprint(c_code)
			c_avp_dict := make_AVPDict(v_map)

			if c_avp_dict.avptype == Avp_Enumerated {
				if v_map["enumarated"] != nil {
					c_avp_enum := v_map["enumarated"].(map[string]interface{})
					for c_enum_key, c_enum_val := range c_avp_enum {
						if c_enum_key == "" {
							continue
						}
						c_enum_key_int, err := strconv.Atoi(c_enum_key)
						if err != nil {
							l.Error.Printf("Dictionary: %s(%s) cannot convert %s to integer", files[c_index], c_key, c_enum_key)
						}

						c_enum_val_str, ok := c_enum_val.(string)
						if !ok {
							l.Error.Printf("Dictionary: cannot convert %v to string, %s", c_enum_key, c_key)
							continue
						}
						//fmt.Printf("%s %s -> %s\n", c_key, c_enum_key, c_enum_val)
						_, ok = avp_enums[c_key]
						if !ok {
							avp_enums[c_key] = make(map[int32]string)
						}
						avp_enums[c_key][int32(c_enum_key_int)] = c_enum_val_str
					}
				}

			}
			dict[c_key] = c_avp_dict
		}
	}
	//fmt.Println(avp_enums)
	//os.Exit(0)
}

func make_AVPDict(c_row map[string]interface{}) AVPDictEntry {
	c_vendor_id := c_row["vendor-id"].(float64)
	c_code := c_row["code"].(float64)
	c_type := c_row["type"].(string)
	c_name := c_row["name"].(string)

	c_type_to_const := Avp_code_unknown
	switch c_type {
	case "OctetString", "IPFilterRule":
		c_type_to_const = Avp_OctetString
	case "Unsigned32", "AppId", "VendorId":
		c_type_to_const = Avp_Unsigned32
	case "Unsigned64":
		c_type_to_const = Avp_Unsigned64
	case "Integer32":
		c_type_to_const = Avp_Integer32
	case "Integer64":
		c_type_to_const = Avp_Integer64
	case "Float32":
		c_type_to_const = Avp_Float32
	case "Float64":
		c_type_to_const = Avp_Float64
	case "UTF8String", "DiameterURI", "DiameterIdentity":
		c_type_to_const = Avp_UTF8String
	case "Enumerated":
		c_type_to_const = Avp_Enumerated
	case "grouped":
		c_type_to_const = Avp_Grouped
	case "Time":
		c_type_to_const = Avp_Time
	case "Address":
		c_type_to_const = Avp_Address
	case "IPAddress":
		c_type_to_const = Avp_OctetString //Avp_IPAddress

	default:
		fmt.Fprintln(os.Stderr, "unknown avp type in dictionary: "+c_type)
		os.Exit(1)

	}

	return AVPDictEntry{
		code:      uint32(c_code),
		vendor_id: uint32(c_vendor_id),
		name:      c_name,
		avptype:   c_type_to_const,
	}
}

func LookUpAvp(avp_code uint32, vendor_id uint32) AVPDictEntry {
	c_key := fmt.Sprint(vendor_id) + "." + fmt.Sprint(avp_code)
	//l.Trace.Printf("lookup key: %s", c_key)
	c_val, c_ok := dict[c_key]
	if c_ok {
		l.Trace.Println("lookup success:", c_key, c_val)
		return c_val
	}
	l.Trace.Println("lookup failed:", c_key)
	return AVPDictEntry{
		avptype:   Avp_code_unknown,
		code:      avp_code,
		name:      "Unknown",
		vendor_id: vendor_id,
	}
}

func LookUpAvp_Enum(avp_code uint32, vendor_id uint32, c_value int32) (string, bool) {
	c_key := fmt.Sprint(vendor_id) + "." + fmt.Sprint(avp_code)

	c_values, ok := avp_enums[c_key]
	if !ok {
		return "", false
	}

	c_string, ok := c_values[c_value]
	if !ok {
		return "", false
	}

	return c_string, true
}
