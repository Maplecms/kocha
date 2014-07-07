package kocha

import (
	"fmt"
	"net/http"
	"os"
	"reflect"
	"runtime"

	"github.com/joho/godotenv"
	"github.com/naoina/miyabi"
)

const (
	DefaultHttpAddr          = "127.0.0.1:9100"
	DefaultMaxClientBodySize = 1024 * 1024 * 10 // 10MB
	StaticDir                = "public"
)

var (
	// Global logger
	Log *Logger
)

// Run starts Kocha app.
func Run(config *Config) error {
	app, err := New(config)
	if err != nil {
		return err
	}
	pid := os.Getpid()
	miyabi.ServerState = func(state miyabi.State) {
		switch state {
		case miyabi.StateStart:
			fmt.Printf("Listening on %s\n", app.Config.Addr)
			fmt.Printf("Server PID: %d\n", pid)
		case miyabi.StateRestart:
			Log.Warn("graceful restarted")
		case miyabi.StateShutdown:
			Log.Warn("graceful shutdown")
		}
	}
	server := &miyabi.Server{
		Addr:    config.Addr,
		Handler: app,
	}
	return server.ListenAndServe()
}

// Application represents a Kocha app.
type Application struct {
	// Config is a configuration of an application.
	Config *Config

	// Router is an HTTP request router of an application.
	Router *Router

	// Template is template sets of an application.
	Template *Template

	// ResourceSet is set of resource of an application.
	ResourceSet ResourceSet
}

// New returns a new Application that configured by config.
func New(config *Config) (*Application, error) {
	app := &Application{Config: config}
	if app.Config.Addr == "" {
		config.Addr = DefaultHttpAddr
	}
	if app.Config.MaxClientBodySize < 1 {
		config.MaxClientBodySize = DefaultMaxClientBodySize
	}
	if err := app.validateSessionConfig(); err != nil {
		return nil, err
	}
	if err := app.buildResourceSet(); err != nil {
		return nil, err
	}
	if err := app.buildTemplate(); err != nil {
		return nil, err
	}
	if err := app.buildRouter(); err != nil {
		return nil, err
	}
	Log = initLogger(config.Logger)
	return app, nil
}

// ServeHTTP implements http.Handler.ServeHTTP.
func (app *Application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	controller, method, args := app.Router.dispatch(r)
	if controller == nil {
		c := NewErrorController(http.StatusNotFound)
		cValue := reflect.ValueOf(c)
		mValue := reflect.ValueOf(c.Get)
		controller = &cValue
		method = &mValue
		args = []reflect.Value{}
	}
	app.render(w, r, controller, method, args)
}

func (app *Application) buildRouter() error {
	router, err := app.Config.RouteTable.buildRouter()
	if err != nil {
		return err
	}
	app.Router = router
	return nil
}

func (app *Application) buildResourceSet() error {
	app.ResourceSet = app.Config.ResourceSet
	return nil
}

func (app *Application) buildTemplate() error {
	t, err := newTemplate(app).build()
	if err != nil {
		return err
	}
	app.Template = t
	return nil
}

func (app *Application) validateSessionConfig() error {
	for _, m := range app.Config.Middlewares {
		if middleware, ok := m.(*SessionMiddleware); ok {
			if app.Config.Session == nil {
				return fmt.Errorf("Because %T is nil, %T cannot be used", app.Config, *middleware)
			}
			if app.Config.Session.Store == nil {
				return fmt.Errorf("Because %T.Store is nil, %T cannot be used", *app.Config, *middleware)
			}
			return nil
		}
	}
	return app.Config.Session.Validate()
}

func (app *Application) render(w http.ResponseWriter, r *http.Request, controller, method *reflect.Value, args []reflect.Value) {
	request := newRequest(r)
	response := newResponse(w)
	var (
		cc     *Controller
		result []reflect.Value
	)
	defer func() {
		defer func() {
			if err := recover(); err != nil {
				logStackAndError(err)
				response.StatusCode = http.StatusInternalServerError
				http.Error(response, http.StatusText(response.StatusCode), response.StatusCode)
			}
		}()
		if err := recover(); err != nil {
			logStackAndError(err)
			c := NewErrorController(http.StatusInternalServerError)
			if cc == nil {
				cc = &Controller{}
				cc.Request = request
				cc.Response = response
			}
			c.Controller = cc
			r := c.Get()
			result = []reflect.Value{reflect.ValueOf(r)}
		}
		for _, m := range app.Config.Middlewares {
			m.After(app, cc)
		}
		response.WriteHeader(response.StatusCode)
		result[0].Interface().(Result).Proc(response)
	}()
	request.Body = http.MaxBytesReader(w, request.Body, app.Config.MaxClientBodySize)
	ac := controller.Elem()
	ccValue := ac.FieldByName("Controller")
	switch c := ccValue.Interface().(type) {
	case Controller:
		cc = &c
	case *Controller:
		cc = &Controller{}
		ccValue.Set(reflect.ValueOf(cc))
		ccValue = ccValue.Elem()
	default:
		panic(fmt.Errorf("BUG: Controller field must be struct of %T or that pointer, but %T", cc, c))
	}
	if err := request.ParseMultipartForm(app.Config.MaxClientBodySize); err != nil && err != http.ErrNotMultipart {
		panic(err)
	}
	cc.Name = ac.Type().Name()
	cc.Layout = app.Config.DefaultLayout
	cc.Context = Context{}
	cc.Request = request
	cc.Response = response
	cc.Params = newParams(cc, request.Form, "")
	cc.App = app
	for _, m := range app.Config.Middlewares {
		m.Before(app, cc)
	}
	ccValue.Set(reflect.ValueOf(*cc))
	result = method.Call(args)
}

// Config represents a application-scope configuration.
type Config struct {
	Addr              string
	AppPath           string
	AppName           string
	DefaultLayout     string
	TemplateSet       TemplateSet
	RouteTable        RouteTable
	Logger            *Logger
	Middlewares       []Middleware
	Session           *SessionConfig
	MaxClientBodySize int64

	ResourceSet ResourceSet
}

// SettingEnv is similar to os.Getenv.
// However, SettingEnv returns def value if the variable is not present, and
// sets def to environment variable.
func SettingEnv(key, def string) string {
	env := os.Getenv(key)
	if env != "" {
		return env
	}
	os.Setenv(key, def)
	return def
}

func logStackAndError(err interface{}) {
	buf := make([]byte, 4096)
	runtime.Stack(buf, false)
	Log.Error("%v\n%v", err, string(buf))
}

func init() {
	_ = godotenv.Load()
}
