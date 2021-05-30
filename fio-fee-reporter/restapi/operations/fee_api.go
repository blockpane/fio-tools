// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/runtime/security"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// NewFeeAPI creates a new Fee instance
func NewFeeAPI(spec *loads.Document) *FeeAPI {
	return &FeeAPI{
		handlers:            make(map[string]map[string]http.Handler),
		formats:             strfmt.Default,
		defaultConsumes:     "application/json",
		defaultProduces:     "application/json",
		customConsumers:     make(map[string]runtime.Consumer),
		customProducers:     make(map[string]runtime.Producer),
		PreServerShutdown:   func() {},
		ServerShutdown:      func() {},
		spec:                spec,
		useSwaggerUI:        false,
		ServeError:          errors.ServeError,
		BasicAuthenticator:  security.BasicAuth,
		APIKeyAuthenticator: security.APIKeyAuth,
		BearerAuthenticator: security.BearerAuth,

		JSONConsumer: runtime.JSONConsumer(),

		JSONProducer: runtime.JSONProducer(),

		GetFeeHandler: GetFeeHandlerFunc(func(params GetFeeParams) middleware.Responder {
			return middleware.NotImplemented("operation GetFee has not yet been implemented")
		}),
		GetFeeByActionNameHandler: GetFeeByActionNameHandlerFunc(func(params GetFeeByActionNameParams) middleware.Responder {
			return middleware.NotImplemented("operation GetFeeByActionName has not yet been implemented")
		}),
		GetFeeByActionNameUsdHandler: GetFeeByActionNameUsdHandlerFunc(func(params GetFeeByActionNameUsdParams) middleware.Responder {
			return middleware.NotImplemented("operation GetFeeByActionNameUsd has not yet been implemented")
		}),
		GetFeeUsdHandler: GetFeeUsdHandlerFunc(func(params GetFeeUsdParams) middleware.Responder {
			return middleware.NotImplemented("operation GetFeeUsd has not yet been implemented")
		}),
		GetFeeVotesFeevoteProducerHandler: GetFeeVotesFeevoteProducerHandlerFunc(func(params GetFeeVotesFeevoteProducerParams) middleware.Responder {
			return middleware.NotImplemented("operation GetFeeVotesFeevoteProducer has not yet been implemented")
		}),
		GetFeeVotesMultiplierProducerHandler: GetFeeVotesMultiplierProducerHandlerFunc(func(params GetFeeVotesMultiplierProducerParams) middleware.Responder {
			return middleware.NotImplemented("operation GetFeeVotesMultiplierProducer has not yet been implemented")
		}),
		GetFeeVotesProducerHandler: GetFeeVotesProducerHandlerFunc(func(params GetFeeVotesProducerParams) middleware.Responder {
			return middleware.NotImplemented("operation GetFeeVotesProducer has not yet been implemented")
		}),
		GetFeeVotesProducerUsdHandler: GetFeeVotesProducerUsdHandlerFunc(func(params GetFeeVotesProducerUsdParams) middleware.Responder {
			return middleware.NotImplemented("operation GetFeeVotesProducerUsd has not yet been implemented")
		}),
		GetPriceHandler: GetPriceHandlerFunc(func(params GetPriceParams) middleware.Responder {
			return middleware.NotImplemented("operation GetPrice has not yet been implemented")
		}),
	}
}

/*FeeAPI Provides information about FIO fees */
type FeeAPI struct {
	spec            *loads.Document
	context         *middleware.Context
	handlers        map[string]map[string]http.Handler
	formats         strfmt.Registry
	customConsumers map[string]runtime.Consumer
	customProducers map[string]runtime.Producer
	defaultConsumes string
	defaultProduces string
	Middleware      func(middleware.Builder) http.Handler
	useSwaggerUI    bool

	// BasicAuthenticator generates a runtime.Authenticator from the supplied basic auth function.
	// It has a default implementation in the security package, however you can replace it for your particular usage.
	BasicAuthenticator func(security.UserPassAuthentication) runtime.Authenticator

	// APIKeyAuthenticator generates a runtime.Authenticator from the supplied token auth function.
	// It has a default implementation in the security package, however you can replace it for your particular usage.
	APIKeyAuthenticator func(string, string, security.TokenAuthentication) runtime.Authenticator

	// BearerAuthenticator generates a runtime.Authenticator from the supplied bearer token auth function.
	// It has a default implementation in the security package, however you can replace it for your particular usage.
	BearerAuthenticator func(string, security.ScopedTokenAuthentication) runtime.Authenticator

	// JSONConsumer registers a consumer for the following mime types:
	//   - application/json
	JSONConsumer runtime.Consumer

	// JSONProducer registers a producer for the following mime types:
	//   - application/json
	JSONProducer runtime.Producer

	// GetFeeHandler sets the operation handler for the get fee operation
	GetFeeHandler GetFeeHandler
	// GetFeeByActionNameHandler sets the operation handler for the get fee by action name operation
	GetFeeByActionNameHandler GetFeeByActionNameHandler
	// GetFeeByActionNameUsdHandler sets the operation handler for the get fee by action name usd operation
	GetFeeByActionNameUsdHandler GetFeeByActionNameUsdHandler
	// GetFeeUsdHandler sets the operation handler for the get fee usd operation
	GetFeeUsdHandler GetFeeUsdHandler
	// GetFeeVotesFeevoteProducerHandler sets the operation handler for the get fee votes feevote producer operation
	GetFeeVotesFeevoteProducerHandler GetFeeVotesFeevoteProducerHandler
	// GetFeeVotesMultiplierProducerHandler sets the operation handler for the get fee votes multiplier producer operation
	GetFeeVotesMultiplierProducerHandler GetFeeVotesMultiplierProducerHandler
	// GetFeeVotesProducerHandler sets the operation handler for the get fee votes producer operation
	GetFeeVotesProducerHandler GetFeeVotesProducerHandler
	// GetFeeVotesProducerUsdHandler sets the operation handler for the get fee votes producer usd operation
	GetFeeVotesProducerUsdHandler GetFeeVotesProducerUsdHandler
	// GetPriceHandler sets the operation handler for the get price operation
	GetPriceHandler GetPriceHandler

	// ServeError is called when an error is received, there is a default handler
	// but you can set your own with this
	ServeError func(http.ResponseWriter, *http.Request, error)

	// PreServerShutdown is called before the HTTP(S) server is shutdown
	// This allows for custom functions to get executed before the HTTP(S) server stops accepting traffic
	PreServerShutdown func()

	// ServerShutdown is called when the HTTP(S) server is shut down and done
	// handling all active connections and does not accept connections any more
	ServerShutdown func()

	// Custom command line argument groups with their descriptions
	CommandLineOptionsGroups []swag.CommandLineOptionsGroup

	// User defined logger function.
	Logger func(string, ...interface{})
}

// UseRedoc for documentation at /docs
func (o *FeeAPI) UseRedoc() {
	o.useSwaggerUI = false
}

// UseSwaggerUI for documentation at /docs
func (o *FeeAPI) UseSwaggerUI() {
	o.useSwaggerUI = true
}

// SetDefaultProduces sets the default produces media type
func (o *FeeAPI) SetDefaultProduces(mediaType string) {
	o.defaultProduces = mediaType
}

// SetDefaultConsumes returns the default consumes media type
func (o *FeeAPI) SetDefaultConsumes(mediaType string) {
	o.defaultConsumes = mediaType
}

// SetSpec sets a spec that will be served for the clients.
func (o *FeeAPI) SetSpec(spec *loads.Document) {
	o.spec = spec
}

// DefaultProduces returns the default produces media type
func (o *FeeAPI) DefaultProduces() string {
	return o.defaultProduces
}

// DefaultConsumes returns the default consumes media type
func (o *FeeAPI) DefaultConsumes() string {
	return o.defaultConsumes
}

// Formats returns the registered string formats
func (o *FeeAPI) Formats() strfmt.Registry {
	return o.formats
}

// RegisterFormat registers a custom format validator
func (o *FeeAPI) RegisterFormat(name string, format strfmt.Format, validator strfmt.Validator) {
	o.formats.Add(name, format, validator)
}

// Validate validates the registrations in the FeeAPI
func (o *FeeAPI) Validate() error {
	var unregistered []string

	if o.JSONConsumer == nil {
		unregistered = append(unregistered, "JSONConsumer")
	}

	if o.JSONProducer == nil {
		unregistered = append(unregistered, "JSONProducer")
	}

	if o.GetFeeHandler == nil {
		unregistered = append(unregistered, "GetFeeHandler")
	}
	if o.GetFeeByActionNameHandler == nil {
		unregistered = append(unregistered, "GetFeeByActionNameHandler")
	}
	if o.GetFeeByActionNameUsdHandler == nil {
		unregistered = append(unregistered, "GetFeeByActionNameUsdHandler")
	}
	if o.GetFeeUsdHandler == nil {
		unregistered = append(unregistered, "GetFeeUsdHandler")
	}
	if o.GetFeeVotesFeevoteProducerHandler == nil {
		unregistered = append(unregistered, "GetFeeVotesFeevoteProducerHandler")
	}
	if o.GetFeeVotesMultiplierProducerHandler == nil {
		unregistered = append(unregistered, "GetFeeVotesMultiplierProducerHandler")
	}
	if o.GetFeeVotesProducerHandler == nil {
		unregistered = append(unregistered, "GetFeeVotesProducerHandler")
	}
	if o.GetFeeVotesProducerUsdHandler == nil {
		unregistered = append(unregistered, "GetFeeVotesProducerUsdHandler")
	}
	if o.GetPriceHandler == nil {
		unregistered = append(unregistered, "GetPriceHandler")
	}

	if len(unregistered) > 0 {
		return fmt.Errorf("missing registration: %s", strings.Join(unregistered, ", "))
	}

	return nil
}

// ServeErrorFor gets a error handler for a given operation id
func (o *FeeAPI) ServeErrorFor(operationID string) func(http.ResponseWriter, *http.Request, error) {
	return o.ServeError
}

// AuthenticatorsFor gets the authenticators for the specified security schemes
func (o *FeeAPI) AuthenticatorsFor(schemes map[string]spec.SecurityScheme) map[string]runtime.Authenticator {
	return nil
}

// Authorizer returns the registered authorizer
func (o *FeeAPI) Authorizer() runtime.Authorizer {
	return nil
}

// ConsumersFor gets the consumers for the specified media types.
// MIME type parameters are ignored here.
func (o *FeeAPI) ConsumersFor(mediaTypes []string) map[string]runtime.Consumer {
	result := make(map[string]runtime.Consumer, len(mediaTypes))
	for _, mt := range mediaTypes {
		switch mt {
		case "application/json":
			result["application/json"] = o.JSONConsumer
		}

		if c, ok := o.customConsumers[mt]; ok {
			result[mt] = c
		}
	}
	return result
}

// ProducersFor gets the producers for the specified media types.
// MIME type parameters are ignored here.
func (o *FeeAPI) ProducersFor(mediaTypes []string) map[string]runtime.Producer {
	result := make(map[string]runtime.Producer, len(mediaTypes))
	for _, mt := range mediaTypes {
		switch mt {
		case "application/json":
			result["application/json"] = o.JSONProducer
		}

		if p, ok := o.customProducers[mt]; ok {
			result[mt] = p
		}
	}
	return result
}

// HandlerFor gets a http.Handler for the provided operation method and path
func (o *FeeAPI) HandlerFor(method, path string) (http.Handler, bool) {
	if o.handlers == nil {
		return nil, false
	}
	um := strings.ToUpper(method)
	if _, ok := o.handlers[um]; !ok {
		return nil, false
	}
	if path == "/" {
		path = ""
	}
	h, ok := o.handlers[um][path]
	return h, ok
}

// Context returns the middleware context for the fee API
func (o *FeeAPI) Context() *middleware.Context {
	if o.context == nil {
		o.context = middleware.NewRoutableContext(o.spec, o, nil)
	}

	return o.context
}

func (o *FeeAPI) initHandlerCache() {
	o.Context() // don't care about the result, just that the initialization happened
	if o.handlers == nil {
		o.handlers = make(map[string]map[string]http.Handler)
	}

	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/fee"] = NewGetFee(o.context, o.GetFeeHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/fee/byActionName"] = NewGetFeeByActionName(o.context, o.GetFeeByActionNameHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/fee/byActionName/usd"] = NewGetFeeByActionNameUsd(o.context, o.GetFeeByActionNameUsdHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/fee/usd"] = NewGetFeeUsd(o.context, o.GetFeeUsdHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/fee/votes/feevote/{producer}"] = NewGetFeeVotesFeevoteProducer(o.context, o.GetFeeVotesFeevoteProducerHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/fee/votes/multiplier/{producer}"] = NewGetFeeVotesMultiplierProducer(o.context, o.GetFeeVotesMultiplierProducerHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/fee/votes/{producer}"] = NewGetFeeVotesProducer(o.context, o.GetFeeVotesProducerHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/fee/votes/{producer}/usd"] = NewGetFeeVotesProducerUsd(o.context, o.GetFeeVotesProducerUsdHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/price"] = NewGetPrice(o.context, o.GetPriceHandler)
}

// Serve creates a http handler to serve the API over HTTP
// can be used directly in http.ListenAndServe(":8000", api.Serve(nil))
func (o *FeeAPI) Serve(builder middleware.Builder) http.Handler {
	o.Init()

	if o.Middleware != nil {
		return o.Middleware(builder)
	}
	if o.useSwaggerUI {
		return o.context.APIHandlerSwaggerUI(builder)
	}
	return o.context.APIHandler(builder)
}

// Init allows you to just initialize the handler cache, you can then recompose the middleware as you see fit
func (o *FeeAPI) Init() {
	if len(o.handlers) == 0 {
		o.initHandlerCache()
	}
}

// RegisterConsumer allows you to add (or override) a consumer for a media type.
func (o *FeeAPI) RegisterConsumer(mediaType string, consumer runtime.Consumer) {
	o.customConsumers[mediaType] = consumer
}

// RegisterProducer allows you to add (or override) a producer for a media type.
func (o *FeeAPI) RegisterProducer(mediaType string, producer runtime.Producer) {
	o.customProducers[mediaType] = producer
}

// AddMiddlewareFor adds a http middleware to existing handler
func (o *FeeAPI) AddMiddlewareFor(method, path string, builder middleware.Builder) {
	um := strings.ToUpper(method)
	if path == "/" {
		path = ""
	}
	o.Init()
	if h, ok := o.handlers[um][path]; ok {
		o.handlers[method][path] = builder(h)
	}
}
