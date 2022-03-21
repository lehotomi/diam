package templates

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	d "github.com/lehotomi/diam/diam"
	l "github.com/lehotomi/diam/mlog"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

const ()

const (
	VAL_FIX                           = 1
	VAL_PARAM                         = 2
	VAL_PARAM_WITH_DEFAULT            = 3
	VAL_ACTION                        = 4
	VAL_PARAM_WITH_ACTION             = 5
	VAL_PARAM_WITH_DEFAULT_AND_ACTION = 6
)

const (
	RET_STRING = 0
	RET_TIME   = 1
)

const (
	const_date_format = "2006-01-02 15:04:05 MST"
)

type TemplRow struct {
	level          int
	vendor_id      uint32
	avp_code       uint32
	mandatory_flag bool
	avp_type       int
	valueType      int
	value          string
	defValue       string
	action         string
}

type header_info map[string]string

var templates map[string][]TemplRow = make(map[string][]TemplRow)

var template_headers map[string]header_info = make(map[string]header_info)

func IsTemplateExists(c_template string) bool {
	_, ok := templates[c_template]
	if ok {
		return true
	}
	return false
}

func FillTemplate(c_template string, pars map[string]string) (d.Message, error) {

	c_templ, ok := templates[c_template]
	if !ok {
		err_str := fmt.Sprintf("template %s does not exist", c_template)
		l.Error.Println(err_str)
		return d.Message{}, errors.New(err_str)
	}

	avps := collectRow(0, c_templ, pars)
	c_header := template_headers[c_template]
	c_cmd_code := stringToAvp_Unsigned32(c_header["command_code"])
	c_app_id := stringToAvp_Unsigned32(c_header["application_id"])

	c_request := false
	if c_header["request"] == "1" {
		c_request = true
	}

	c_proxiable := false
	if c_header["proxiable"] == "1" {
		c_proxiable = true
	}

	ret := d.GenMess(c_cmd_code, c_request, c_proxiable, c_app_id, 0, 0, avps)
	return ret, nil
}

func computeValue(row TemplRow, pars map[string]string) (interface{}, int) {

	switch row.valueType {
	case VAL_FIX:
		return row.value, RET_STRING
	case VAL_PARAM:
		val, ok := pars[row.value]
		if ok {
			return val, RET_STRING
		}
		l.Warn.Println("unable to find value for parameter ", row.value)
		//TODO
		return val, RET_STRING
	case VAL_PARAM_WITH_DEFAULT:
		val, ok := pars[row.value]
		if ok {
			return val, RET_STRING
		} else {
			return row.defValue, RET_STRING
		}
	case VAL_ACTION:
		//if (row.action == "now" ) {
		//t := time.Now()
		//l.Info.Println("action!!!",row,row.action)
		ret_val := runAction("", row.action)
		return ret_val, RET_STRING
		//return t.Format(const_date_format),RET_STRING
		//}

	case VAL_PARAM_WITH_ACTION:
		val, ok := pars[row.value]
		if ok {
			//l.Info.Println("running action on:", val, row.action)
			mod_val := runAction(val, row.action)
			return mod_val, RET_STRING
		} else {
			return row.defValue, RET_STRING
		}
	case VAL_PARAM_WITH_DEFAULT_AND_ACTION:

		val, ok := pars[row.value]

		if !ok {
			val = row.defValue
		}

		//l.Info.Println("running action on 2:", val, row.action)
		mod_val := runAction(val, row.action)
		return mod_val, RET_STRING
	default:
		l.Warn.Println("unknown valueType:", row.valueType)
		return string('0'), RET_STRING
	}

}

func runAction(inp string, action string) string {

	switch action {
	case "mccnmc_to_user_loc":
		if len(inp) < 5 {
			l.Warn.Println("mccnmc is invalid:", inp)
			return "0012f607ffffffff"
		}
		return "00" + string(inp[1]) + string(inp[0]) + "f" + string(inp[2]) + string(inp[4]) + string(inp[3]) + "ffffffff"

	case "now":
		t := time.Now()
		//l.Info.Println("action!!!",row)
		return t.Format(const_date_format)
	}
	l.Warn.Println("unknown action in template:", action)
	return "0"
}

func collectRow(level int, rows []TemplRow, pars map[string]string) []d.AVP {
	var ret []d.AVP

	for j := 0; j < len(rows); j++ {
		if rows[j].level == level {
			res_value, ret_type := computeValue(rows[j], pars)
			var computed_value string
			if ret_type == RET_STRING {
				computed_value = res_value.(string)
			}

			switch rows[j].avp_type {
			case d.Avp_Unsigned32:
				ret = append(ret,
					d.AVP_Unsigned32(
						rows[j].avp_code,
						stringToAvp_Unsigned32(computed_value),
						rows[j].mandatory_flag,
						rows[j].vendor_id))
			case d.Avp_Unsigned64:
				ret = append(ret,
					d.AVP_Unsigned64(
						rows[j].avp_code,
						stringToAvp_Unsigned64(computed_value),
						rows[j].mandatory_flag,
						rows[j].vendor_id))
			case d.Avp_Integer32, d.Avp_Enumerated:
				ret = append(ret,
					d.AVP_Integer32(
						rows[j].avp_code,
						stringToAvp_Integer32(computed_value),
						rows[j].mandatory_flag,
						rows[j].vendor_id))
			case d.Avp_Integer64:
				ret = append(ret,
					d.AVP_Integer64(
						rows[j].avp_code,
						stringToAvp_Integer64(computed_value),
						rows[j].mandatory_flag,
						rows[j].vendor_id))
			case d.Avp_Float32:
				ret = append(ret,
					d.AVP_Float32(
						rows[j].avp_code,
						stringToAvp_Float32(computed_value),
						rows[j].mandatory_flag,
						rows[j].vendor_id))
			case d.Avp_Float64:
				ret = append(ret,
					d.AVP_Float64(
						rows[j].avp_code,
						stringToAvp_Float64(computed_value),
						rows[j].mandatory_flag,
						rows[j].vendor_id))
			case d.Avp_UTF8String:
				ret = append(ret,
					d.AVP_UTF8String(
						rows[j].avp_code,
						computed_value,
						rows[j].mandatory_flag,
						rows[j].vendor_id))
			case d.Avp_OctetString:
				ret = append(ret,
					d.AVP_OctetString(
						rows[j].avp_code,
						stringToAvp_Octetstring(computed_value),
						rows[j].mandatory_flag,
						rows[j].vendor_id))
			case d.Avp_Time:
				ret = append(ret,
					d.AVP_Time(
						rows[j].avp_code,
						stringToAvp_Time(computed_value),
						rows[j].mandatory_flag,
						rows[j].vendor_id))
			case d.Avp_Address:
				ret = append(ret,
					d.AVP_Address(
						rows[j].avp_code,
						stringToAvp_Address(computed_value),
						rows[j].mandatory_flag,
						rows[j].vendor_id))
			case d.Avp_IPAddress:
				ret = append(ret,
					d.AVP_OctetString(
						rows[j].avp_code,
						IPToAvp_Octetstring(computed_value),
						rows[j].mandatory_flag,
						rows[j].vendor_id))
			case d.Avp_Grouped:
				var s int
				for s = j + 1; s < len(rows); s++ {
					if rows[s].level == level {
						break
					}
				}
				group_row := rows[j+1 : s]
				ret = append(ret,
					d.AVP_Group(
						rows[j].avp_code,
						collectRow(level+1, group_row, pars),
						rows[j].mandatory_flag,
						rows[j].vendor_id),
				)
			default:
				l.Error.Printf("Unknown AVP type: %d", rows[j].avp_type)
				ret = append(ret,
					d.AVP_Unsigned32(
						rows[j].avp_code,
						stringToAvp_Unsigned32(computed_value),
						rows[j].mandatory_flag,
						rows[j].vendor_id))
			}
		}
	}

	return ret
}

func stringToAvp_Unsigned32(in string) uint32 {
	res, err := strconv.ParseUint(in, 10, 32)
	if err != nil {
		l.Warn.Printf("cannot convert %s to uint32", in)
		return 9999
	}
	return uint32(res)
}

func stringToAvp_Integer32(in string) int32 {
	res, err := strconv.ParseInt(in, 10, 32)
	if err != nil {
		l.Warn.Printf("cannot convert %s to int32", in)
		return 9999
	}
	return int32(res)
}

func stringToAvp_Float32(in string) float32 {
	res, err := strconv.ParseFloat(in, 32)
	if err != nil {
		l.Warn.Printf("cannot convert %s to float32", in)
		return 9999
	}
	return float32(res)
}

func stringToAvp_Unsigned64(in string) uint64 {
	res, err := strconv.ParseUint(in, 10, 64)
	if err != nil {
		l.Warn.Printf("cannot convert %s to uint64", in)
		return 9999
	}
	return res
}

func stringToAvp_Integer64(in string) int64 {
	res, err := strconv.ParseInt(in, 10, 64)
	if err != nil {
		l.Warn.Printf("cannot convert %s to int64", in)
		return 9999
	}
	return int64(res)
}

func stringToAvp_Float64(in string) float64 {
	res, err := strconv.ParseFloat(in, 64)
	if err != nil {
		l.Warn.Printf("cannot convert %s to float64", in)
		return 9999
	}
	return res
}

func stringToAvp_Time(in string) time.Time {

	//format := "2006-01-02 15:04:05 MST"
	res, err := time.Parse(const_date_format, in)
	if err != nil {
		l.Warn.Printf("cannot convert %s to Time", in)
		return time.Now()
	}
	return res
}

func IPToAvp_Octetstring(in string) []byte {

	// todo check if ipv6, and convert that
	return d.IPv4ToByte(in)
}
func stringToAvp_Octetstring(in string) []byte {

	res, err := hex.DecodeString(in)
	if err != nil {
		l.Warn.Printf("cannot convert %s to OctetString", in)
		return []byte{0x00}
	}
	return res
}

func stringToAvp_Address(in string) d.Address {
	//TODO check ip
	res := d.NewAddress(d.ENUM_ADDR_FAMILY, d.IPv4ToByte(in))
	return res
}

func Init(templ_dir string) {
	//l.init()
	var files []string
	var file_cont [][]string
	fileInfo, err := ioutil.ReadDir(templ_dir) //TODO
	if err != nil {
		fmt.Println("cannot find template dir ...", templ_dir)
		os.Exit(1)
		return
	}

	for _, file := range fileInfo {
		c_file := file.Name()
		if strings.HasSuffix(c_file, ".template") {
			files = append(files, c_file)
		}
		//files = append(files, file.Name())
	}
	l.Trace.Println("template files:", files)
	for _, c_file := range files {
		var c_temps []TemplRow
		c_tempfile, err := os.Open(templ_dir + "/" + c_file)
		c_file_parts := strings.Split(c_file, ".")
		c_file_basename := strings.Join(c_file_parts[:len(c_file_parts)-1], ".")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer c_tempfile.Close()
		//byte_temp, _ := ioutil.ReadAll(c_tempfile)
		var lines []string
		var c_has_header bool = false
		scanner := bufio.NewScanner(c_tempfile)
		header_params := make(map[string]string)

		for scanner.Scan() {
			c_line := scanner.Text()
			c_line = adjustLine(c_line)
			if c_line == "" {
				continue
			}
			tab_index := strings.Index(c_line, "\t")
			if tab_index != -1 {
				l.Error.Printf("%s contains tab, at position %d, line:'%s'", c_file, tab_index, c_line)
				os.Exit(1)
			}
			num_of_leading_sp := countLeadingSpaces(c_line)

			if num_of_leading_sp%4 != 0 {
				l.Error.Printf("%s should have 0, 4, 8, 12, ... leading spaces, line:'%s'", c_file, c_line)
				os.Exit(1)
			}

			if strings.HasPrefix(c_line, "!header ") {
				c_has_header = true

				if ind_spaces := strings.Index(c_line, "  "); ind_spaces != -1 {
					l.Error.Printf("%s header should only contain one space as separator, at postition %d, line:'%s'", c_file, ind_spaces, c_line)
					os.Exit(1)
				}
				c_line := strings.TrimPrefix(c_line, "!header ")
				c_head_parts := strings.Split(c_line, " ")
				for _, v := range c_head_parts {
					//fmt.Println("_"+v+"_")
					c_name_value := strings.Split(v, ":")
					if len(c_name_value) != 2 {
						l.Error.Printf("%s header is not valid: %s", c_file, c_line)
						os.Exit(1)
					}
					header_params[c_name_value[0]] = c_name_value[1]
				}

				template_headers[c_file_basename] = header_params
				continue
			}

			level := num_of_leading_sp / 4
			c_line = c_line[num_of_leading_sp:]
			c_line = fmt.Sprintf("%d.%s", level, c_line)

			more_than_one_space_index := strings.Index(c_line, "  ")
			first_i := strings.Index(c_line, "'")

			if more_than_one_space_index != -1 && more_than_one_space_index < first_i {
				l.Error.Printf("%s should only contain one space as separator, at postition %d, line:'%s'", c_file, more_than_one_space_index, c_line)
				os.Exit(1)
			}

			c_line_parts := strings.Split(c_line, " ")

			c_avp_type := d.AvpStringToConst(c_line_parts[1])

			if c_avp_type == -1 {
				l.Error.Printf("%s, avp type not known:'%s'", c_file, c_line_parts[1])
				os.Exit(1)
			}

			if c_line_parts[1] != "Grouped" {
				first_i := strings.Index(c_line, "'")
				last_i := strings.LastIndex(c_line, "'")

				if first_i == -1 || last_i == -1 || first_i == last_i {
					l.Error.Printf("%s, avp value should be between ', line:\"%s\"", c_file, c_line)
					os.Exit(1)
				}
				c_line_parts[2] = c_line[first_i+1 : last_i]
				c_line_parts = c_line_parts[0:3]
			}

			c_line_parts[1] = fmt.Sprintf("%d", c_avp_type)
			c_line = strings.Join(c_line_parts, ".")
			lines = append(lines, c_line)

			c_line_split := strings.Split(c_line, ".")

			c_level, err := strconv.ParseInt(c_line_split[0], 10, 32)
			if err != nil {
				l.Error.Printf("cannot parse %s as integer, line: %s", c_line_split[0], c_line)
				os.Exit(1)
			}

			c_vendor_id, err := strconv.ParseUint(c_line_split[1], 10, 32)
			if err != nil {
				l.Error.Printf("cannot parse %s as integer, line: %s", c_line_split[1], c_line)
				os.Exit(1)
			}
			c_avp_code, err := strconv.ParseUint(c_line_split[2], 10, 32)
			if err != nil {
				l.Error.Printf("cannot parse %s as integer, line: %s", c_line_split[2], c_line)
				os.Exit(1)
			}
			c_mand_flag := true
			if c_line_split[3] == "0" {
				c_mand_flag = false
			}

			c_value := "NA"
			if c_avp_type != d.Avp_Grouped {
				if len(c_line_split) < 6 {
					l.Error.Printf("avp value missing, line: %s", c_line)
					os.Exit(1)
				}
				c_value = strings.Join(c_line_split[5:], ".")
			} else {
				c_value = "_grouped_"
			}

			c_new_trow := makeTmplRow(int(c_level), uint32(c_vendor_id), uint32(c_avp_code), c_mand_flag, c_avp_type, c_value)
			c_temps = append(c_temps, c_new_trow)

		} // every line

		for j, b := range c_temps {
			if j == 0 && b.level != 0 {
				l.Error.Printf("first template avp sohuld not have space prefix")
				os.Exit(1)
			}
			if j == 0 {
				continue
			}
			if (b.level == c_temps[j-1].level+1) && c_temps[j-1].avp_type != d.Avp_Grouped {
				l.Error.Printf("misaligned row: %s", c_file)
				os.Exit(1)
			}

		}
		templates[c_file_basename] = c_temps
		if c_has_header == false {
			l.Error.Printf("%s, header is not defined", c_file)
			os.Exit(1)
		}

		file_cont = append(file_cont, lines)
	} //every file

}

func makeTmplRow(c_level int, c_vendor_id uint32, c_avp_code uint32, c_mand_flag bool, c_avp_type int, c_value string) TemplRow {

	ret := TemplRow{
		level:     c_level,
		vendor_id: c_vendor_id,

		avp_code:       c_avp_code,
		mandatory_flag: c_mand_flag,
		avp_type:       c_avp_type,
		/*valueType int,*/
		value: c_value,
	}

	c_str_value := ret.value

	if !strings.HasPrefix(c_str_value, "{{") {
		ret.valueType = VAL_FIX
	} else {

		if strings.HasPrefix(c_str_value, "{{!") {
			ret.valueType = VAL_ACTION
			ret.action = c_str_value[3 : len(c_str_value)-2]
		} else {
			c_stripped_value := c_str_value[2 : len(c_str_value)-2]
			c_colon_pos := strings.Index(c_stripped_value, ":")
			c_excl_pos := strings.Index(c_stripped_value, "!")
			if (c_colon_pos == -1) && (c_excl_pos == -1) {
				ret.value = c_stripped_value
				ret.valueType = VAL_PARAM
			} else if (c_colon_pos != -1) && (c_excl_pos != -1) {
				//10415.22.1 OctetString '{{user_location:21670!mccnmc_to_user_loc}}'

				ret.value = c_stripped_value[:c_colon_pos]
				ret.action = c_stripped_value[c_excl_pos+1:]
				ret.defValue = c_stripped_value[c_colon_pos+1 : c_excl_pos]
				ret.valueType = VAL_PARAM_WITH_DEFAULT_AND_ACTION
				//fmt.Println("VAL_PARAM_WITH_DEFAULT_AND_ACTION:",ret)

			} else if c_colon_pos != -1 {
				ret.value = c_stripped_value[:c_colon_pos]
				ret.defValue = c_stripped_value[c_colon_pos+1:]
				ret.valueType = VAL_PARAM_WITH_DEFAULT
			} else {
				ret.value = c_stripped_value[:c_excl_pos]
				ret.action = c_stripped_value[c_excl_pos+1:]
				ret.defValue = "0"
				ret.valueType = VAL_PARAM_WITH_ACTION
			}

		}
	}
	return ret
}

func countLeadingSpaces(line string) int {
	return len(line) - len(strings.TrimLeft(line, " "))
}

func adjustLine(line string) string {
	line = stripComment(line)
	line = strings.TrimRight(line, " \t")
	return line
}

func stripComment(source string) string {
	if cut := strings.Index(source, "#"); cut >= 0 {
		return source[:cut]
	}
	return source
}
