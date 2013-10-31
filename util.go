package kocha

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"text/template"
	"unicode"
)

func ToCamelCase(s string) string {
	result := make([]rune, 0, len(s))
	upper := false
	for _, r := range s {
		if r == '_' {
			upper = true
			continue
		}
		if upper {
			result = append(result, unicode.ToUpper(r))
			upper = false
			continue
		}
		result = append(result, r)
	}
	result[0] = unicode.ToUpper(result[0])
	return string(result)
}

func ToSnakeCase(s string) string {
	var result bytes.Buffer
	result.WriteRune(unicode.ToLower(rune(s[0])))
	for _, c := range s[1:] {
		if unicode.IsUpper(c) {
			result.WriteRune('_')
		}
		result.WriteRune(unicode.ToLower(c))
	}
	return result.String()
}

func normPath(p string) string {
	result := path.Clean(p)
	// path.Clean() truncate the trailing slash but add it.
	if p[len(p)-1] == '/' && result != "/" {
		result += "/"
	}
	return result
}

type Error struct {
	Usager  usager
	Message string
}

func (e Error) Error() string {
	return e.Message
}

type usager interface {
	Usage() string
}

type fileStatus uint8

const (
	fileStatusConflict fileStatus = iota
	fileStatusNoConflict
	fileStatusIdentical
)

func PanicOnError(usager usager, format string, a ...interface{}) {
	panic(Error{usager, fmt.Sprintf(format, a...)})
}

func CopyTemplate(u usager, srcPath, dstPath string, data map[string]interface{}) {
	tmpl, err := template.ParseFiles(srcPath)
	if err != nil {
		PanicOnError(u, "abort: failed to parse template: %v", err)
	}
	var bufFrom bytes.Buffer
	if err := tmpl.Execute(&bufFrom, data); err != nil {
		PanicOnError(u, "abort: failed to process template: %v", err)
	}
	printFunc := PrintCreate
	switch detectConflict(u, bufFrom.Bytes(), dstPath) {
	case fileStatusConflict:
		PrintConflict(dstPath)
		if !confirmOverwrite(dstPath) {
			PrintSkip(dstPath)
			return
		}
		printFunc = PrintOverwrite
	case fileStatusIdentical:
		PrintIdentical(dstPath)
		return
	}
	dstFile, err := os.Create(dstPath)
	if err != nil {
		PanicOnError(u, "abort: failed to create file: %v", err)
	}
	defer dstFile.Close()
	if _, err := io.Copy(dstFile, &bufFrom); err != nil {
		PanicOnError(u, "abort: failed to output file: %v", err)
	}
	printFunc(dstPath)
}

func detectConflict(u usager, src []byte, dstPath string) fileStatus {
	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		return fileStatusNoConflict
	}
	dstBuf, err := ioutil.ReadFile(dstPath)
	if err != nil {
		PanicOnError(u, "abort: failed to read file: %v", err)
	}
	if bytes.Equal(src, dstBuf) {
		return fileStatusIdentical
	}
	return fileStatusConflict
}

func confirmOverwrite(dstPath string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("Overwrite %v? [Yn] ", dstPath)
		yesno, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		switch strings.ToUpper(strings.TrimSpace(yesno)) {
		case "", "YES", "Y":
			return true
		case "NO", "N":
			return false
		}
	}
}

type colorfunc func(string, ...interface{}) string

func Red(s string, a ...interface{}) string {
	return color(31, s, a...)
}

func Green(s string, a ...interface{}) string {
	return color(32, s, a...)
}

func Yellow(s string, a ...interface{}) string {
	return color(33, s, a...)
}

func Blue(s string, a ...interface{}) string {
	return color(34, s, a...)
}

func Magenta(s string, a ...interface{}) string {
	return color(35, s, a...)
}

func Cyan(s string, a ...interface{}) string {
	return color(36, s, a...)
}

func color(colorCode int, s string, a ...interface{}) string {
	switch length := len(a); {
	case length == 0:
		a = append(a, "%s")
	case length > 1:
		panic(errors.New("too many arguments"))
	}
	return fmt.Sprintf(fmt.Sprintf("\x1b[%d;1m%s\x1b[0m", colorCode, a[0]), s)
}

func PrintIdentical(path string) {
	printPathStatus(Blue, "identical", path)
}

func PrintConflict(path string) {
	printPathStatus(Red, "conflict", path)
}

func PrintSkip(path string) {
	printPathStatus(Cyan, "skip", path)
}

func PrintOverwrite(path string) {
	printPathStatus(Cyan, "overwrite", path)
}

func PrintCreate(path string) {
	printPathStatus(Green, "create", path)
}

func PrintExist(path string) {
	printPathStatus(Blue, "exist", path)
}

func PrintCreateDirectory(path string) {
	printPathStatus(Green, "create directory", path)
}

func printPathStatus(f colorfunc, message, s string) {
	fmt.Println(f(message, "%20s"), "", s)
}
