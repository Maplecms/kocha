package kocha

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/naoina/kocha/util"
)

// Render returns result of template.
//
// A data to used will be determined the according to the following rules.
//
// 1. If data of the Data type is given, it will be merged to Context.Data.
//
// 2. If data of another type is given, it will be set to Context.Data.
//
// 3. If data is nil, Context.Data as is.
//
// Render retrieve a template file from controller name and c.Response.ContentType.
// e.g. If controller name is "root" and ContentType is "application/xml", Render will
// try to retrieve the template file "root.xml".
// Also ContentType set to "text/html" if not specified.
func Render(c *Context, data interface{}) error {
	c.setData(data)
	c.setContentTypeIfNotExists("text/html")
	if err := c.setFormatFromContentTypeIfNotExists(); err != nil {
		return err
	}
	t, err := c.App.Template.Get(c.App.Config.AppName, c.Layout, c.Name, c.Format)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, c); err != nil {
		return err
	}
	return c.render(&buf)
}

// RenderJSON returns result of JSON.
//
// RenderJSON is similar to Render but data will be encoded to JSON.
// ContentType set to "application/json" if not specified.
func RenderJSON(c *Context, data interface{}) error {
	c.setData(data)
	c.setContentTypeIfNotExists("application/json")
	buf, err := json.Marshal(c.Data)
	if err != nil {
		return err
	}
	return c.render(bytes.NewReader(buf))
}

// RenderXML returns result of XML.
//
// RenderXML is similar to Render but data will be encoded to XML.
// ContentType set to "application/xml" if not specified.
func RenderXML(c *Context, data interface{}) error {
	c.setData(data)
	c.setContentTypeIfNotExists("application/xml")
	buf, err := xml.Marshal(c.Data)
	if err != nil {
		return err
	}
	return c.render(bytes.NewReader(buf))
}

// RenderText returns result of text.
//
// ContentType set to "text/plain" if not specified.
func RenderText(c *Context, content string) error {
	c.setContentTypeIfNotExists("text/plain")
	return c.render(strings.NewReader(content))
}

// RenderError returns result of error.
//
// RenderError is similar to Render, but there is a point where some different.
// Render retrieve a template file from statusCode and c.Response.ContentType.
// e.g. If statusCode is 500 and ContentType is "application/xml", Render will
// try to retrieve the template file "errors/500.xml".
// If failed to retrieve the template file, it returns result of text with statusCode.
// Also ContentType set to "text/html" if not specified.
func RenderError(c *Context, statusCode int, data interface{}) error {
	c.setData(data)
	c.setContentTypeIfNotExists("text/html")
	if err := c.setFormatFromContentTypeIfNotExists(); err != nil {
		return err
	}
	c.Response.StatusCode = statusCode
	c.Name = errorTemplateName(statusCode)
	t, err := c.App.Template.Get(c.App.Config.AppName, c.Layout, c.Name, c.Format)
	if err != nil {
		c.Response.ContentType = "text/plain"
		return c.render(bytes.NewReader([]byte(http.StatusText(statusCode))))
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, c); err != nil {
		return err
	}
	return c.render(&buf)
}

// SendFile returns result of any content.
//
// The path argument specifies an absolute or relative path.
// If absolute path, read the content from the path as it is.
// If relative path, First, Try to get the content from included resources and
// returns it if successful. Otherwise, Add AppPath and StaticDir to the prefix
// of the path and then will read the content from the path that.
// Also, set ContentType detect from content if c.Response.ContentType is empty.
func SendFile(c *Context, path string) error {
	var file io.ReadSeeker
	path = filepath.FromSlash(path)
	if rc := c.App.ResourceSet.Get(path); rc != nil {
		switch b := rc.(type) {
		case string:
			file = strings.NewReader(b)
		case []byte:
			file = bytes.NewReader(b)
		}
	}
	if file == nil {
		if !filepath.IsAbs(path) {
			path = filepath.Join(c.App.Config.AppPath, StaticDir, path)
		}
		if _, err := os.Stat(path); err != nil {
			return RenderError(c, http.StatusNotFound, nil)
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		file = f
	}
	c.Response.ContentType = util.DetectContentTypeByExt(path)
	if c.Response.ContentType == "" {
		c.Response.ContentType = util.DetectContentTypeByBody(file)
	}
	return c.render(file)
}

// Redirect returns result of redirect.
//
// If permanently is true, redirect to url with 301. (http.StatusMovedPermanently)
// Otherwise redirect to url with 302. (http.StatusFound)
func Redirect(c *Context, url string, permanently bool) error {
	if permanently {
		c.Response.StatusCode = http.StatusMovedPermanently
	} else {
		c.Response.StatusCode = http.StatusFound
	}
	http.Redirect(c.Response, c.Request.Request, url, c.Response.StatusCode)
	return nil
}
