package kocha

import (
	"errors"
	"fmt"
	"net/url"
)

type mimeTypeFormats map[string]string

var MimeTypeFormats = mimeTypeFormats{
	"application/json": "json",
	"application/xml":  "xml",
	"text/html":        "html",
	"text/plain":       "txt",
}

func (m mimeTypeFormats) Get(mimeType string) string {
	return m[mimeType]
}

func (m mimeTypeFormats) Set(mimeType, format string) {
	m[mimeType] = format
}

func (m mimeTypeFormats) Del(mimeType string) {
	delete(m, mimeType)
}

type Controller struct {
	Name     string
	Request  *Request
	Response *Response
	Params   Params
}

type Context map[string]interface{}

func (c *Controller) Render(context ...Context) Result {
	var ctx Context
	switch len(context) {
	case 0: // do nothing
	case 1:
		ctx = context[0]
	default: // > 1
		panic(errors.New("too many arguments"))
	}
	format := MimeTypeFormats.Get(c.Response.ContentType)
	if format == "" {
		panic(fmt.Errorf("unknown Content-Type: %v", c.Response.ContentType))
	}
	t := appConfig.TemplateSet.Get(appConfig.AppName, c.Name, format)
	if t == nil {
		panic(errors.New("no such template: " + appConfig.TemplateSet.Ident(appConfig.AppName, c.Name, format)))
	}
	return &ResultTemplate{
		Template: t,
		Context:  ctx,
	}
}

func (c *Controller) RenderJSON(context interface{}) Result {
	return &ResultJSON{
		Context: context,
	}
}

func (c *Controller) RenderXML(context interface{}) Result {
	return &ResultXML{
		Context: context,
	}
}

func (c *Controller) RenderPlainText(content string) Result {
	return &ResultPlainText{
		Content: content,
	}
}

type Params struct {
	url.Values
}
