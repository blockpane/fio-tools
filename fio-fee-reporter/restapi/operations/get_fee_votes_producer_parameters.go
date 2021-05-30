// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
)

// NewGetFeeVotesProducerParams creates a new GetFeeVotesProducerParams object
//
// There are no default values defined in the spec.
func NewGetFeeVotesProducerParams() GetFeeVotesProducerParams {

	return GetFeeVotesProducerParams{}
}

// GetFeeVotesProducerParams contains all the bound params for the get fee votes producer operation
// typically these are obtained from a http.Request
//
// swagger:parameters GetFeeVotesProducer
type GetFeeVotesProducerParams struct {

	// HTTP Request Object
	HTTPRequest *http.Request `json:"-"`

	/*The producer's account
	  Required: true
	  In: path
	*/
	Producer string
}

// BindRequest both binds and validates a request, it assumes that complex things implement a Validatable(strfmt.Registry) error interface
// for simple values it will use straight method calls.
//
// To ensure default values, the struct must have been initialized with NewGetFeeVotesProducerParams() beforehand.
func (o *GetFeeVotesProducerParams) BindRequest(r *http.Request, route *middleware.MatchedRoute) error {
	var res []error

	o.HTTPRequest = r

	rProducer, rhkProducer, _ := route.Params.GetOK("producer")
	if err := o.bindProducer(rProducer, rhkProducer, route.Formats); err != nil {
		res = append(res, err)
	}
	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// bindProducer binds and validates parameter Producer from path.
func (o *GetFeeVotesProducerParams) bindProducer(rawData []string, hasKey bool, formats strfmt.Registry) error {
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: true
	// Parameter is provided by construction from the route
	o.Producer = raw

	return nil
}
