package core

import (
	"bytes"
	"context"
	"crypto/sha1"
	b64 "encoding/base64"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	tmpl "text/template"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/dgraph-io/badger/v3"
	"github.com/gabriel-vasile/mimetype"
	"github.com/google/renameio"
	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/approle"
	"github.com/itchyny/gojq"
	log "github.com/livelace/logrus"
	"github.com/spf13/viper"
)

var (
	TemplateFuncMap = tmpl.FuncMap{
		"FromBase64": Base64Decode,
		"ToEscape":   JsonEscape,
		"ToBase64":   Base64Encode,
		"ToLower":    strings.ToLower,
		"ToUpper":    strings.ToUpper,
	}
)

func Base64Decode(s string) (string, error) {
	result, err := b64.StdEncoding.DecodeString(s)
	if err != nil {
		return fmt.Sprintf("decode error: %s", err), err
	}

	return fmt.Sprintf("%s", result), err
}

func Base64Encode(s string) string {
	return b64.StdEncoding.EncodeToString([]byte(s))
}

func BytesToSize(s int64) string {
	return bytefmt.ByteSize(uint64(s))
}

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

func GetStringFromStringSlice(s *[]string) string {
	r := ""
	for _, v := range *s {
		r += v
	}
	return r
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

func ExtractDatumFieldIntoArray(data *Datum, field interface{}) []string {
	temp := make([]string, 0)

	// "field" might be just a regular string (not data field)
	// or string representation of a data field.
	// Example (both are strings):
	// "qwerty" - regular string.
	// "rss.title" - string representation of data field.
	//
	// All data field might be expanded (reflect) into string, int, slice etc.
	// All expanded data appended to []string.

	// Try to reflect string into data field.
	f := func(v string) []string {
		t := make([]string, 0)

		rv, err := ReflectDatumField(data, v)

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

func ExtractDatumFieldIntoString(data *Datum, field interface{}) string {
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

		rv, err := ReflectDatumField(data, v)

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

func ExtractJqQueriesIntoArray(config *viper.Viper, queries []string) [][]*gojq.Query {
	temp := make([][]*gojq.Query, 0)

	for _, q := range queries {
		currentQueries := make([]*gojq.Query, 0)
		templateQueries := config.GetStringSlice(fmt.Sprintf("%s.query", q))

		if len(templateQueries) > 0 {
			for _, v := range templateQueries {
				query, err := gojq.Parse(v)
				if err == nil {
					currentQueries = append(currentQueries, query)
				}
			}
		} else {
			query, err := gojq.Parse(q)
			if err == nil {
				currentQueries = append(currentQueries, query)
			}
		}

		temp = append(temp, currentQueries)
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

func ExtractTemplateIntoString(i *Datum, t *tmpl.Template) (string, error) {
	var b bytes.Buffer

	if err := t.Execute(&b, i); err != nil {
		return "", err
	}

	return b.String(), nil
}

func ExtractTemplateMapIntoStringMap(i *Datum, m map[string]*tmpl.Template) (map[string]string, error) {
	result := make(map[string]string, len(m))

	for k, t := range m {
		s, err := ExtractTemplateIntoString(i, t)
		if err != nil {
			return result, err
		}
		result[k] = s
	}

	return result, nil
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

func GenUID() string {
	runes := []rune("abcdefghijklmnopqrstuvwxyz1234567890")

	rand.Seed(time.Now().UnixNano())

	b := make([]rune, 6)
	for i := range b {
		b[i] = runes[rand.Intn(len(runes))]
	}
	return string(b)
}

func GetCredValue(cred string, vault *vault.Client) string {
	// Get from environment variable.
	if strings.Contains(cred, "env://") {
		return GetVarFromEnv(cred)
	}

	// Get from vault secret.
	if c := strings.Split(cred, "vault://"); len(c) > 1 && vault != nil {
		if s := strings.Split(c[1], ","); len(s) > 1 {
			secretPath := s[0]
			secretKey := s[1]

			secret, err := vault.Logical().Read(secretPath)
			if err != nil {
				return ""
			}

			value, ok := secret.Data[secretKey].(string)
			if !ok {
				return ""
			}

			return value
		}
	}

	return cred
}

func GetVarFromEnv(v string) string {
	if c := strings.Split(v, "env://"); len(c) > 1 {
		return os.Getenv(c[1])
	}
	return v
}

func GetDatumFieldType(field interface{}) (reflect.Kind, error) {
	if f, ok := field.(string); ok {
		rv, err := ReflectDatumField(&Datum{}, f)

		if err != nil {
			return 0, fmt.Errorf(ERROR_DATA_FIELD_UNKNOWN.Error(), field)
		} else {
			return rv.Kind(), nil
		}

	} else {
		return 0, fmt.Errorf(ERROR_DATA_FIELD_UNKNOWN.Error(), field)
	}
}

func GetFileMimeType(file string) (*mimetype.MIME, error) {
    if _, err := IsFile(file); err == nil {
		return mimetype.DetectFile(file)
	} else {
		return &mimetype.MIME{}, err
	}
}

func GetFileNameAndExtension(file string) (string, string) {
	if file == "/" || file == " " || file == "" || file == "." || file == ".." {
		return "", ""
	}

	fileWithExtension := filepath.Base(file)
	fileExtension := filepath.Ext(fileWithExtension)
	fileName := fileWithExtension[0 : len(fileWithExtension)-len(fileExtension)]

	return fileName, fileExtension
}

func GetVault(m map[string]interface{}) (*vault.Client, error) {
	if len(m) > 0 {
		var address string
		var app_role string
		var app_secret string

		if v, b := IsString(m["address"]); b {
			if address = GetVarFromEnv(v); address == "" {
				return nil, fmt.Errorf("vault address env not set: %v", v)
			}
		} else {
			return nil, fmt.Errorf("vault address must be set: %v", m["address"])
		}

		if v, b := IsString(m["app_role"]); b {
			if app_role = GetVarFromEnv(v); app_role == "" {
				return nil, fmt.Errorf("vault app_role env not set: %v", v)
			}
		} else {
			return nil, fmt.Errorf("vault app_role must be set: %v", m["app_role"])
		}

		if v, b := IsString(m["app_secret"]); b {
			if app_secret = GetVarFromEnv(v); app_secret == "" {
				return nil, fmt.Errorf("vault app_secret env not set: %v", v)
			}
		} else {
			return nil, fmt.Errorf("vault app_secret must be set: %v", m["app_secret"])
		}

		vaultConfig := vault.DefaultConfig()
		vaultConfig.Address = address

		vaultClient, err := vault.NewClient(vaultConfig)
		if err != nil {
			return nil, fmt.Errorf("cannot initialize vault client: %v", err)
		}

		appRoleAuth, err := auth.NewAppRoleAuth(
			app_role,
			&auth.SecretID{FromString: app_secret},
		)
		if err != nil {
			return nil, fmt.Errorf("cannot initialize app role auth: %v", err)
		}

		authInfo, err := vaultClient.Auth().Login(context.TODO(), appRoleAuth)
		if err != nil {
			return nil, fmt.Errorf("cannot login with app role: %v", err)
		}
		if authInfo == nil {
			return nil, fmt.Errorf("no auth info for app role")
		}

		return vaultClient, nil
	}

	return nil, nil
}

func HashString(s *string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(*s)))
}

func IntervalToMilliseconds(s string) (int64, error) {
	digitsPattern := regexp.MustCompile("[0-9]+")
	formatPattern := regexp.MustCompile("^[0-9]+[SMHD]+$")

	su := strings.ToUpper(s)

	if !formatPattern.MatchString(strings.ToUpper(s)) {
		return 0, fmt.Errorf("%s: %s", ERROR_INTERVAL_FORMAT_UNKNOWN, s)
	} else {
		v, _ := strconv.ParseInt(string(digitsPattern.Find([]byte(s))), 10, 64)

		// cannot be zero, cast to 1.
		if v == 0 {
			v = 1
		}

		switch {
		case strings.HasSuffix(su, "MS"):
			return v, nil
		case strings.HasSuffix(su, "S"):
			return v * 1000, nil
		case strings.HasSuffix(su, "M"):
			return v * 60 * 1000, nil
		case strings.HasSuffix(su, "H"):
			return v * 60 * 60 * 1000, nil
		case strings.HasSuffix(su, "D"):
			return v * 24 * 60 * 60 * 1000, nil
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
		if strings.Contains(b, "env://") {
			b = GetVarFromEnv(b)
		}
		if v, err := strconv.ParseBool(fmt.Sprintf("%v", b)); err == nil {
			return v, true
		}
	}
	return false, false
}

func IsChatUsername(i interface{}) (string, bool) {
	re := regexp.MustCompile("^@[a-zA-Z0-9.\\-_]+$")

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

	if err != nil {
		return false
	}

	return info.IsDir()
}

func IsDatumFieldsSlice(fields *[]string) error {
	temp := make([]string, 0)

	for _, field := range *fields {
		rv, err := ReflectDatumField(&Datum{}, field)

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

func IsDatumFieldsTypesEqual(a *[]string, b *[]string) error {
	temp := make([]string, 0)

	for i := 0; i < len(*a); i++ {
		ra, ea := ReflectDatumField(&Datum{}, (*a)[i])
		rb, eb := ReflectDatumField(&Datum{}, (*b)[i])

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

func IsFile(file string) (time.Time, error) {
	info, err := os.Stat(file)
	if err != nil || info.IsDir() || !info.Mode().IsRegular() {
		return time.Now().UTC(), fmt.Errorf(ERROR_FILE_INVALID.Error(), file)
	}
	return info.ModTime(), nil
}

func IsFlowNameValid(name string) bool {
	re := regexp.MustCompile("^[a-zA-Z0-9-]+$")
	return re.MatchString(name)
}

func IsFloat(i interface{}, args ...bool) (float32, bool) {
	canBeZero := false
	if len(args) > 0 {
		canBeZero = args[0]
	}

	switch v := i.(type) {
	case int:
		if canBeZero || (!canBeZero && v > 0) {
			return float32(v), true
		}
	case float64:
		if canBeZero || (!canBeZero && v > 0) {
			return float32(v), true
		}
	case string:
		if strings.Contains(v, "env://") {
			v = GetVarFromEnv(v)
		}
		if sf, err := strconv.ParseFloat(v, 64); err == nil && sf > 0 {
			return float32(sf), true
		}
	}
	return 0, false
}

func IsInt(i interface{}, args ...bool) (int, bool) {
	canBeZero := false
	if len(args) > 0 {
		canBeZero = args[0]
	}

	switch v := i.(type) {
	case int:
		if canBeZero || (!canBeZero && v > 0) {
			return v, true
		}
	case string:
		if strings.Contains(v, "env://") {
			v = GetVarFromEnv(v)
		}
		if si, err := strconv.ParseInt(v, 10, 64); err == nil && si > 0 {
			return int(si), true
		}
	}
	return 0, false
}

func IsInterval(i interface{}) (int64, bool) {
	if v, b := IsString(i); b {
		ms, err := IntervalToMilliseconds(v)

		if err != nil {
			return 0, false
		} else {
			return ms, true
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

func IsSliceOfSliceInt(i interface{}) ([][]int, bool) {
	temp := make([][]int, 0)

	if i == nil {
		return temp, false

	} else if ii, ok := i.([]interface{}); ok {
		if len(ii) == 0 {
			return temp, false
		}

		for _, v := range ii {
			i, ok := IsSliceOfInt(v)

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
		if strings.Contains(v, "env://") {
			v = GetVarFromEnv(v)
		}
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

func JsonEscape(s string) (string, error) {
	result, err := json.Marshal(s)
	if err != nil {
		return fmt.Sprintf("escape error: %s", err), err
	}

	return string(result[1 : len(result)-1]), err
}

func MapKeysToStringSlice(m *map[string]interface{}) []string {
	temp := make([]string, 0)

	for k := range *m {
		temp = append(temp, k)
	}

	sort.Strings(temp)

	return temp
}

func PluginLoadData(database string, data interface{}) error {
    if _, err := IsFile(database); err == nil {
		// open file.
		f, err := os.OpenFile(database, os.O_RDONLY, 0644)
		if err != nil {
			return fmt.Errorf(ERROR_PLUGIN_LOAD_DATA.Error(), err)
		}

		// get file size.
		fs, err := f.Stat()
		if err != nil {
			return fmt.Errorf(ERROR_PLUGIN_LOAD_DATA.Error(), err)
		}

		// read file.
		content := make([]byte, fs.Size())
		_, err = f.Read(content)
		if err != nil {
			return fmt.Errorf(ERROR_PLUGIN_LOAD_DATA.Error(), err)
		}

		// decode data.
		decoder := gob.NewDecoder(bytes.NewReader(content))
		err = decoder.Decode(data)
		if err != nil {
			return fmt.Errorf(ERROR_PLUGIN_LOAD_DATA.Error(), err)
		}

		err = f.Close()

	} else {
        return err
    }

	return nil
}

func LogInputPlugin(fields log.Fields, source string, message interface{}) {
	f := log.Fields{}
	for k, v := range fields {
		f[k] = v
	}

	_, ok := message.(error)

	f["source"] = source

	if ok {
		f["error"] = fmt.Sprintf("%v", message)
		log.WithFields(f).Error(LOG_PLUGIN_DATA)

	} else {
		f["data"] = fmt.Sprintf("%v", message)
		log.WithFields(f).Debug(LOG_PLUGIN_DATA)
	}
}

func LogProcessPlugin(fields log.Fields, message interface{}) {
	f := log.Fields{}
	for k, v := range fields {
		f[k] = v
	}

	_, ok := message.(error)

	if ok {
		f["error"] = fmt.Sprintf("%v", message)
		log.WithFields(f).Error(LOG_PLUGIN_DATA)

	} else {
		f["data"] = fmt.Sprintf("%v", message)
		log.WithFields(f).Debug(LOG_PLUGIN_DATA)
	}
}

func LogOutputPlugin(fields log.Fields, destination string, message interface{}) {
	f := log.Fields{}
	for k, v := range fields {
		f[k] = v
	}

	_, ok := message.(error)

	f["destination"] = destination

	if ok {
		f["error"] = fmt.Sprintf("%v", message)
		log.WithFields(f).Error(LOG_PLUGIN_DATA)

	} else {
		f["data"] = fmt.Sprintf("%v", message)
		log.WithFields(f).Debug(LOG_PLUGIN_DATA)
	}
}

func SymlinkFile(source string, destination string) error {
	return renameio.Symlink(source, destination)
}

func PluginLoadState(database string, data *map[string]time.Time) error {
	// Disable logging.
	opts := badger.DefaultOptions(database)
	opts.Logger = nil

	// Open database.
	db, err := badger.Open(opts)
	if err != nil {
		return err
	}
	defer db.Close()

	// Read database.
	err = db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 100

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()

			err := item.Value(func(value []byte) error {
				signature := fmt.Sprintf("%s", item.Key())
				timestamp, err := time.Parse(time.RFC3339, fmt.Sprintf("%s", value))
				(*data)[signature] = timestamp

				return err
			})

			if err != nil {
				return err
			}
		}
		return nil
	})

	return err
}

func PluginSaveData(database string, data interface{}) error {
	buffer := new(bytes.Buffer)
	encoder := gob.NewEncoder(buffer)

	err := encoder.Encode(data)
	if err != nil {
		return fmt.Errorf(ERROR_PLUGIN_SAVE_DATA.Error(), err)
	}

	err = renameio.WriteFile(database, buffer.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf(ERROR_PLUGIN_SAVE_DATA.Error(), err)
	}

	return nil
}

func PluginSaveState(database string, data *map[string]time.Time, ttl time.Duration) error {
	// Disable logging.
	opts := badger.DefaultOptions(database)
	opts.Logger = nil
	opts.SyncWrites = true

	// Open database.
	db, err := badger.Open(opts)
	if err != nil {
		return err
	}
	defer db.Close()

	// Save data.
	txn := db.NewTransaction(true)
	for signature, timestamp := range *data {
		e := badger.NewEntry([]byte(signature), []byte(timestamp.Format(time.RFC3339))).WithTTL(ttl)
		if err := txn.SetEntry(e); err == badger.ErrTxnTooBig {
			_ = txn.Commit()
			txn = db.NewTransaction(true)
			_ = txn.Set([]byte(signature), []byte(timestamp.Format(time.RFC3339)))
		}
	}
	if err = txn.Commit(); err != nil {
		return err
	}

	// Garbage collection.
	err = db.RunValueLogGC(0.5)
	if err == badger.ErrNoRewrite {
		return nil
	}

	return err
}

func ReflectDatumField(item *Datum, i interface{}) (reflect.Value, error) {
	var temp reflect.Value

	// Datum field key must be string.
	field, ok := i.(string)
	if !ok {
		return temp, fmt.Errorf(ERROR_DATA_FIELD_KEY.Error(), i)
	}

	// Datum fields might be:
	// 1. <Datum>.<FirstLevel>: Datum.Time
	// 2. <Datum>.<FirstLevel>.<SecondLevel>: Datum.RSS.TITLE
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

func ShowPluginParam(fields log.Fields, key string, value interface{}) {
	f := log.Fields{}
	for k, v := range fields {
		f[k] = v
	}

	f["value"] = fmt.Sprintf("%s: %v", key, value)
	log.WithFields(f).Debug(LOG_SET_VALUE)
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
		"time", "level", "msg", "path", "hash", "run", "flow", "instance", "file", "plugin",
		"type", "source", "destination", "id", "alias", "include", "value", "data", "error", "message",
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

	f, err := os.OpenFile(fp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)

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

func GetStringFromFile(file string) (string, error) {
    if _, err := IsFile(file); err != nil {
		return "", err
	}

	b, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}

    return string(b), nil
}

func GetLinesFromFile(file string) ([]string, error) {
    s, err := GetStringFromFile(file)
    if err != nil {
        return make([]string, 0), err
    }
    return strings.Split(s, "\n"), nil
}
