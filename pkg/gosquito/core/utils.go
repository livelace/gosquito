package core

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"github.com/gabriel-vasile/mimetype"
	"github.com/spf13/viper"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	tmpl "text/template"
	"time"
)

var (
	TemplateFuncMap = tmpl.FuncMap{
		"ToLower": strings.ToLower,
		"ToUpper": strings.ToUpper,
	}
)

func CheckPluginParams(availableParams *map[string]int, params *map[string]interface{}) error {
	paramsRequired := make([]string, 0)
	paramsUnknown := make([]string, 0)

	// Check for strictly required parameters.
	for k, v := range *availableParams {
		if v > 0 {
			paramsRequired = append(paramsRequired, k)
		}
	}

	if len(paramsRequired) > 0 {
		return fmt.Errorf(ERROR_PLUGIN_REQUIRED_PARAM.Error(), paramsRequired)
	}

	// Check for unknown parameters.
	for k := range *params {
		if _, ok := (*availableParams)[k]; !ok {
			paramsUnknown = append(paramsUnknown, k)
		}
	}

	if len(paramsUnknown) > 0 {
		return fmt.Errorf(ERROR_PARAM_UNKNOWN.Error(), paramsUnknown)
	}

	return nil
}

func CreateDirIfNotExist(d string) error {
	if !IsDir(d) {
		if err := os.MkdirAll(d, os.FileMode(0755)); err != nil {
			return err
		}
	}

	return nil
}

func DetectFileType(path string) (*mimetype.MIME, error) {
	if IsFile(path, "") {
		return mimetype.DetectFile(path)
	} else {
		return &mimetype.MIME{}, fmt.Errorf("not a file: %v", path)
	}
}

func ExtractConfigVariableIntoArray(config *viper.Viper, variable interface{}) []string {
	temp := make([]string, 0)

	f := func(v string) []string {
		t := make([]string, 0)

		rv := reflect.ValueOf(config.Get(v))

		switch rv.Kind() {
		case reflect.String:
			t = append(t, fmt.Sprintf("%v", v))

		case reflect.Slice:
			for i := 0; i < rv.Len(); i++ {
				t = append(t, fmt.Sprintf("%v", rv.Index(i)))
			}
		default:
			t = append(t, v)
		}

		return t
	}

	if v, ok := variable.(string); ok {
		temp = append(temp, f(v)...)

	} else if arr, ok := variable.([]string); ok {
		for _, v := range arr {
			temp = append(temp, f(v)...)
		}
	}

	return temp
}

func ExtractDataFieldIntoArray(data *DataItem, field interface{}) []string {
	temp := make([]string, 0)

	// "field" might be just a regular string (not data field)
	// or string representation of a data field.
	// Example (both are strings):
	// "qwerty" - regular string.
	// "rss.title" - string representation of data field.
	//
	// Every data field might be expanded (reflect) into string, int, slice etc.
	// All expanded data appended to []string.

	// Try to reflect string into data field.
	f := func(v string) []string {
		t := make([]string, 0)

		rv, err := ReflectDataField(data, v)

		if err != nil {
			t = append(t, fmt.Sprintf("%v", v))
		} else {
			if rv.Kind() == reflect.Slice {
				for i := 0; i < rv.Len(); i++ {
					t = append(t, fmt.Sprintf("%v", rv.Index(i)))
				}

			} else {
				t = append(t, fmt.Sprintf("%v", rv))
			}
		}

		return t
	}

	// Check provided field.
	if v, ok := field.(string); ok {
		temp = append(temp, f(v)...)

	} else if arr, ok := field.([]string); ok {
		for _, v := range arr {
			temp = append(temp, f(v)...)
		}

	} else if v, ok := field.(interface{}); ok {
		rv := reflect.ValueOf(v)

		switch rv.Kind() {
		case reflect.String:
			temp = append(temp, f(rv.String())...)

		case reflect.Slice:
			for i := 0; i < rv.Len(); i++ {
				temp = append(temp, f(rv.Index(i).Elem().String())...)
			}

		default:
			temp = append(temp, fmt.Sprintf("%v", v))
		}
	}

	return temp
}

func ExtractDataFieldIntoString(data *DataItem, field interface{}) string {
	temp := ""

	// "field" might be just a regular string (not data field)
	// or string representation of a data field.
	// Example (both are strings):
	// "qwerty" - regular string.
	// "rss.title" - string representation of data field.
	//
	// Every data field might be expanded (reflect) into string, int, slice etc.
	// All expanded data appended to string.

	// Try to reflect string into data field.
	f := func(v string) string {
		t := ""

		rv, err := ReflectDataField(data, v)

		if err != nil {
			t += fmt.Sprintf(" %v", v)
		} else {
			if rv.Kind() == reflect.Slice {
				for i := 0; i < rv.Len(); i++ {
					t += fmt.Sprintf(" %v", rv.Index(i))
				}

			} else {
				t += fmt.Sprintf(" %v", rv)
			}
		}

		return t
	}

	// Check provided field.
	if v, ok := field.(string); ok {
		temp += f(v)

	} else if arr, ok := field.([]string); ok {
		for _, v := range arr {
			temp += f(v)
		}

	} else if v, ok := field.(interface{}); ok {
		rv := reflect.ValueOf(v)

		switch rv.Kind() {
		case reflect.String:
			temp += fmt.Sprintf(" %v", f(rv.String()))

		case reflect.Slice:
			for i := 0; i < rv.Len(); i++ {
				temp += fmt.Sprintf(" %v", f(rv.Index(i).Elem().String()))
			}

		default:
			temp += fmt.Sprintf(" %v", v)
		}
	}

	return temp
}

func ExtractRegexpsIntoArrays(config *viper.Viper, regexps []string, matchCase bool) [][]*regexp.Regexp {
	temp := make([][]*regexp.Regexp, 0)

	for _, s := range regexps {
		currentRegexps := make([]*regexp.Regexp, 0)
		templateRegexps := config.GetStringSlice(fmt.Sprintf("%s.regexp", s))

		if len(templateRegexps) > 0 {
			for _, r := range templateRegexps {
				if !matchCase {
					r = fmt.Sprintf("(?i)%s", r)
				}

				if re, err := regexp.Compile(r); err == nil {
					currentRegexps = append(currentRegexps, re)
				} else {
					fmt.Println(err)
				}

			}
		} else {
			if !matchCase {
				s = fmt.Sprintf("(?i)%s", s)
			}

			if re, err := regexp.Compile(s); err == nil {
				currentRegexps = append(currentRegexps, re)
			}
		}

		temp = append(temp, currentRegexps)
	}

	return temp
}

func ExtractScripts(config *viper.Viper, scripts []string) []string {
	temp := make([]string, 0)

	for _, s := range scripts {
		script := config.GetString(fmt.Sprintf("%s.script", s))

		if len(script) > 0 {
			temp = append(temp, script)
		} else {
			temp = append(temp, s)
		}
	}

	return temp
}

func ExtractTemplateIntoString(data *DataItem, t *tmpl.Template) (string, error) {
	var b bytes.Buffer

	if err := t.Execute(&b, data); err != nil {
		return "", err
	}

	return b.String(), nil
}

func ExtractXpathsIntoArrays(config *viper.Viper, xpaths []string) [][]string {
	temp := make([][]string, 0)

	for _, s := range xpaths {
		currentXpaths := make([]string, 0)
		templateXpaths := config.GetStringSlice(fmt.Sprintf("%s.xpath", s))

		if len(templateXpaths) > 0 {
			for _, x := range templateXpaths {
				currentXpaths = append(currentXpaths, x)

			}
		} else {
			currentXpaths = append(currentXpaths, s)
		}

		temp = append(temp, currentXpaths)
	}

	return temp
}

func ExecWithTimeout(cmd string, args []string, timeout int) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(timeout)*time.Second)
	defer cancel()

	c := exec.CommandContext(ctx, cmd, args...)
	return c.Output()
}

func GenFlowHash() string {
	runes := []rune("abcdefghijklmnopqrstuvwxyz1234567890")

	rand.Seed(time.Now().UnixNano())

	b := make([]rune, 6)
	for i := range b {
		b[i] = runes[rand.Intn(len(runes))]
	}
	return string(b)
}

func GetDataFieldType(field interface{}) (reflect.Kind, error) {
	if f, ok := field.(string); ok {
		rv, err := ReflectDataField(&DataItem{}, f)

		if err != nil {
			return 0, fmt.Errorf(ERROR_DATA_FIELD_UNKNOWN.Error(), field)
		} else {
			return rv.Kind(), nil
		}

	} else {
		return 0, fmt.Errorf(ERROR_DATA_FIELD_UNKNOWN.Error(), field)
	}
}

func IntervalToSeconds(s string) (int64, error) {
	su := strings.ToUpper(s)
	r := []rune(su)
	re := regexp.MustCompile("^[0-9]+[SMHD]$")

	if !re.MatchString(su) {
		return 0, fmt.Errorf("%s: %s", ERROR_INTERVAL_FORMAT_UNKNOWN, s)
	} else {
		v, _ := strconv.ParseInt(string(r[:len(r)-1]), 10, 64)

		// cannot be zero, cast to 1.
		if v == 0 {
			v = 1
		}

		switch {
		case strings.HasSuffix(su, "S"):
			return v, nil
		case strings.HasSuffix(su, "M"):
			return v * 60, nil
		case strings.HasSuffix(su, "H"):
			return v * 60 * 60, nil
		case strings.HasSuffix(su, "D"):
			return v * 60 * 60 * 24, nil
		default:
			return 0, fmt.Errorf("%s: %s", ERROR_INTERVAL_FORMAT_UNKNOWN, s)
		}
	}
}

func IsBool(i interface{}) (bool, bool) {
	switch b := i.(type) {
	case bool:
		return b, true
	case string:
		if v, err := strconv.ParseBool(b); err == nil {
			return v, true
		}
		return false, false
	default:
		return false, false
	}
}

func IsChatUsername(i interface{}) (string, bool) {
	re := regexp.MustCompile("^@[a-zA-Z0-9.-_]+$")

	if v, b := IsString(i); b {
		if re.MatchString(v) {
			return string([]rune(v)[1:]), true
		} else {
			return v, false
		}
	} else {
		return "", false
	}
}

func IsDir(path string) bool {
	info, err := os.Stat(path)

	if os.IsNotExist(err) {
		return false
	}

	return info.IsDir()
}

func IsDataFieldsSlice(fields *[]string) error {
	temp := make([]string, 0)

	for _, field := range *fields {
		rv, err := ReflectDataField(&DataItem{}, field)

		if err != nil || rv.Kind() != reflect.Slice {
			temp = append(temp, field)
		}
	}

	if len(temp) > 0 {
		return fmt.Errorf(ERROR_DATA_FIELD_NOT_SLICE.Error(), temp)
	} else {
		return nil
	}
}

func IsDataFieldsTypesEqual(a *[]string, b *[]string) error {
	temp := make([]string, 0)

	for i := 0; i < len(*a); i++ {
		ra, ea := ReflectDataField(&DataItem{}, (*a)[i])
		rb, eb := ReflectDataField(&DataItem{}, (*b)[i])

		if ea != nil || eb != nil || ra.Kind() != rb.Kind() {
			temp = append(temp, (*a)[i])
			temp = append(temp, (*b)[i])
			break
		}
	}

	if len(temp) > 0 {
		return fmt.Errorf(ERROR_DATA_FIELD_TYPE_MISMATCH.Error(), temp)
	} else {
		return nil
	}
}

func IsFile(path string, file string) bool {
	info, err := os.Stat(filepath.Join(path, file))

	if os.IsNotExist(err) {
		return false
	}

	if info.IsDir() || !info.Mode().IsRegular() {
		return false
	}

	return true
}

func IsFlowName(name string) bool {
	re := regexp.MustCompile("^[a-zA-Z0-9-]+$")

	return re.MatchString(name)
}

func IsInt(i interface{}) (int, bool) {
	if v, ok := i.(int); ok && v > 0 {
		return v, true
	} else {
		return 0, false
	}
}

func IsInterval(i interface{}) (int64, bool) {
	if v, b := IsString(i); b {
		seconds, err := IntervalToSeconds(v)

		if err != nil {
			return 0, false
		} else {
			return seconds, true
		}

	} else {
		return 0, false
	}
}

func IsMapWithStringAsKey(i interface{}) (map[string]interface{}, bool) {
	temp := make(map[string]interface{}, 0)

	if i == nil {
		return temp, false

	} else if si, ok := i.(map[string]interface{}); ok {
		if len(si) == 0 {
			return temp, false
		} else {
			return si, true
		}

	} else if ii, ok := i.(map[interface{}]interface{}); ok {
		if len(ii) == 0 {
			return temp, false
		}

		for k, v := range ii {
			if ks, ok := k.(string); ok {
				temp[strings.ToLower(ks)] = v
			} else {
				return temp, false
			}
		}

	} else {
		return temp, false
	}

	return temp, true
}

func IsPluginId(i interface{}) (int, bool) {
	if v, ok := i.(int); ok && v >= 0 {
		return v, true
	} else {
		return 0, false
	}
}

func IsSize(i interface{}) (int64, bool) {
	if v, b := IsString(i); b {
		bb, err := SizeToBytes(v)

		if err != nil {
			return 0, false
		} else {
			return bb, true
		}

	} else {
		return 0, false
	}
}

func IsSliceOfInt(i interface{}) ([]int, bool) {
	temp := make([]int, 0)

	if i == nil {
		return temp, false

	} else if ii, ok := i.([]interface{}); ok {
		if len(ii) == 0 {
			return temp, false
		}

		for _, v := range ii {
			i, ok := v.(int)

			if !ok {
				return temp, false
			} else {
				temp = append(temp, i)
			}
		}

	} else {
		return temp, false
	}

	return temp, true
}

func IsSliceOfString(i interface{}) ([]string, bool) {
	temp := make([]string, 0)

	if i == nil {
		return temp, false

	} else if ss, ok := i.([]string); ok {
		if len(ss) == 0 {
			return temp, false
		} else {
			return ss, true
		}

	} else if is, ok := i.([]interface{}); ok {
		if len(is) == 0 {
			return temp, false
		}

		for _, v := range is {
			s, ok := v.(string)

			if !ok {
				return temp, false
			} else {
				temp = append(temp, s)
			}
		}

	} else {
		return temp, false
	}

	return temp, true
}

func IsString(i interface{}) (string, bool) {
	if v, ok := i.(string); ok && len(v) > 0 {
		return v, true
	} else {
		return "", false
	}
}

func IsTimeZone(i interface{}) (*time.Location, bool) {
	if v, b := IsString(i); b {
		loc, err := time.LoadLocation(v)

		if err == nil {
			return loc, true
		} else {
			return nil, false
		}
	} else {
		return nil, false
	}
}

func IsValueInSlice(v string, s *[]string) bool {
	for _, i := range *s {
		if v == i {
			return true
		}
	}
	return false
}

func MapKeysToStringSlice(m *map[string]interface{}) []string {
	temp := make([]string, 0)

	for k := range *m {
		temp = append(temp, k)
	}

	sort.Strings(temp)

	return temp
}

func PluginLoadData(path string, file string, output interface{}) error {
	if IsFile(path, file) {
		// read file.
		f, err := os.OpenFile(filepath.Join(path, file), os.O_RDONLY, 0644)
		if err != nil {
			return fmt.Errorf(ERROR_PLUGIN_DATA_READ.Error(), err)
		}

		fs, err := f.Stat()
		if err != nil {
			return fmt.Errorf(ERROR_PLUGIN_DATA_READ.Error(), err)
		}

		data := make([]byte, fs.Size())
		_, err = f.Read(data)
		if err != nil {
			return fmt.Errorf(ERROR_PLUGIN_DATA_READ.Error(), err)
		}

		// decode data.
		decoder := gob.NewDecoder(bytes.NewReader(data))
		err = decoder.Decode(output)
		if err != nil {
			return fmt.Errorf(ERROR_PLUGIN_DATA_READ.Error(), err)
		}

		err = f.Close()

		return err
	}

	return nil
}

// TODO: Sources' state time has to be truly last despite of concurrency.
// TODO: proc1 may have time 11111.
// TODO: proc2 may have time 22222.
// TODO: proc2 may save state time as 22222 first.
// TODO: proc1 may save state time as 11111 last.
// TODO: "Last time" 11111 isn't what we expect.
// TODO: Should be atomic operation.
func PluginSaveData(path string, file string, data interface{}) error {
	buffer := new(bytes.Buffer)

	//gob.Register(time.Time{})
	encoder := gob.NewEncoder(buffer)

	err := encoder.Encode(data)
	if err != nil {
		return fmt.Errorf(ERROR_PLUGIN_DATA_WRITE.Error(), err)
	}

	f, err := os.OpenFile(filepath.Join(path, file), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf(ERROR_PLUGIN_DATA_WRITE.Error(), err)
	}

	_, err = f.Write(buffer.Bytes())
	if err != nil {
		return fmt.Errorf(ERROR_PLUGIN_DATA_WRITE.Error(), err)
	}

	return nil
}

func ReflectDataField(item *DataItem, i interface{}) (reflect.Value, error) {
	var temp reflect.Value

	// Data field key must be string.
	field, ok := i.(string)
	if !ok {
		return temp, fmt.Errorf(ERROR_DATA_FIELD_KEY.Error(), i)
	}

	// Data fields might be:
	// 1. <Data>.<FirstLevel>: Data.Time
	// 2. <Data>.<FirstLevel>.<SecondLevel>: Data.RSS.TITLE
	// Everything else is wrong.
	f := strings.ToUpper(field)
	p := strings.Split(f, ".")

	if len(p) == 1 {
		temp = reflect.ValueOf(item).Elem().FieldByName(p[0])

		if !temp.IsValid() {
			return temp, fmt.Errorf(ERROR_DATA_FIELD_UNKNOWN.Error(), field)
		}

	} else if len(p) == 2 {
		if reflect.ValueOf(item).Elem().FieldByName(p[0]).IsValid() &&
			reflect.ValueOf(item).Elem().FieldByName(p[0]).FieldByName(p[1]).IsValid() {

			temp = reflect.ValueOf(item).Elem().FieldByName(p[0]).FieldByName(p[1])

		} else {
			return temp, fmt.Errorf(ERROR_DATA_FIELD_UNKNOWN.Error(), field)
		}

	} else {
		return temp, fmt.Errorf(ERROR_DATA_FIELD_UNKNOWN.Error(), field)
	}

	return temp, nil
}

func SizeToBytes(s string) (int64, error) {
	su := strings.ToUpper(s)
	r := []rune(su)
	re := regexp.MustCompile("^[0-9]+[BKMG]$")

	if !re.MatchString(su) {
		return 0, fmt.Errorf("%s: %s", ERROR_SIZE_FORMAT_UNKNOWN, s)
	} else {
		v, _ := strconv.ParseInt(string(r[:len(r)-1]), 10, 64)

		// cannot be zero, cast to 1.
		if v == 0 {
			v = 1
		}

		switch {
		case strings.HasSuffix(su, "B"):
			return v, nil
		case strings.HasSuffix(su, "K"):
			return v * 1024, nil
		case strings.HasSuffix(su, "M"):
			return v * 1024 * 1024, nil
		case strings.HasSuffix(su, "G"):
			return v * 1024 * 1024 * 1024, nil
		default:
			return 0, fmt.Errorf("%s: %s", ERROR_SIZE_FORMAT_UNKNOWN, s)
		}
	}
}

func SliceStringToUpper(s *[]string) {
	for i, v := range *s {
		(*s)[i] = strings.ToUpper(v)
	}
}

func SortLogFields(s []string) {
	// Ordered fields list.
	order := []string{
		"time", "level", "msg", "path", "hash", "flow", "file", "plugin",
		"type", "value", "source", "id", "alias", "data", "error", "message",
	}

	// Mark found fields.
	found := make(map[string]bool, 0)

	for _, v := range s {
		found[v] = true
	}

	// Counter for ordering.
	c := 0

	// Set values according order.
	for _, v := range order {
		if _, ok := found[v]; ok {
			s[c] = v
			c++
		}
	}
}

func ShrinkString(s *string, l int) string {
	r := []rune(*s)

	if len(r) > l {
		return string(r[:l])
	}

	return *s
}

func UniqueSliceValues(s *[]string) []string {
	temp := make([]string, 0)

	m := make(map[string]bool, 0)

	for _, v := range *s {
		if v != "" {
			if _, ok := m[v]; !ok {
				m[v] = true
				temp = append(temp, v)
			}
		}
	}

	return temp
}

func WriteStringToFile(path string, file string, s string) error {
	fp := filepath.Join(path, file)

	f, err := os.OpenFile(fp, os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	if _, err := f.WriteString(s); err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	return nil
}
