// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/blockpane/fio-tools/fio-fee-reporter/models"
)

// GetFeeByActionNameUsdOKCode is the HTTP code returned for type GetFeeByActionNameUsdOK
const GetFeeByActionNameUsdOKCode int = 200

/*GetFeeByActionNameUsdOK An array of prices for each action in USD

swagger:response getFeeByActionNameUsdOK
*/
type GetFeeByActionNameUsdOK struct {

	/*
	  In: Body
	*/
	Payload []*models.Price `json:"body,omitempty"`
}

// NewGetFeeByActionNameUsdOK creates GetFeeByActionNameUsdOK with default headers values
func NewGetFeeByActionNameUsdOK() *GetFeeByActionNameUsdOK {

	return &GetFeeByActionNameUsdOK{}
}

// WithPayload adds the payload to the get fee by action name usd o k response
func (o *GetFeeByActionNameUsdOK) WithPayload(payload []*models.Price) *GetFeeByActionNameUsdOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get fee by action name usd o k response
func (o *GetFeeByActionNameUsdOK) SetPayload(payload []*models.Price) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetFeeByActionNameUsdOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	payload := o.Payload
	if payload == nil {
		// return empty array
		payload = make([]*models.Price, 0, 50)
	}

	if err := producer.Produce(rw, payload); err != nil {
		panic(err) // let the recovery middleware deal with this
	}
}

// GetFeeByActionNameUsdServiceUnavailableCode is the HTTP code returned for type GetFeeByActionNameUsdServiceUnavailable
const GetFeeByActionNameUsdServiceUnavailableCode int = 503

/*GetFeeByActionNameUsdServiceUnavailable Data is stale, has not been updated for several minutes

swagger:response getFeeByActionNameUsdServiceUnavailable
*/
type GetFeeByActionNameUsdServiceUnavailable struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewGetFeeByActionNameUsdServiceUnavailable creates GetFeeByActionNameUsdServiceUnavailable with default headers values
func NewGetFeeByActionNameUsdServiceUnavailable() *GetFeeByActionNameUsdServiceUnavailable {

	return &GetFeeByActionNameUsdServiceUnavailable{}
}

// WithPayload adds the payload to the get fee by action name usd service unavailable response
func (o *GetFeeByActionNameUsdServiceUnavailable) WithPayload(payload *models.Error) *GetFeeByActionNameUsdServiceUnavailable {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get fee by action name usd service unavailable response
func (o *GetFeeByActionNameUsdServiceUnavailable) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetFeeByActionNameUsdServiceUnavailable) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(503)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}
