// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/blockpane/fio-tools/fio-fee-reporter/models"
)

// GetFeeVotesProducerUsdOKCode is the HTTP code returned for type GetFeeVotesProducerUsdOK
const GetFeeVotesProducerUsdOKCode int = 200

/*GetFeeVotesProducerUsdOK An array of prices for each action in USD based on the producer's votes

swagger:response getFeeVotesProducerUsdOK
*/
type GetFeeVotesProducerUsdOK struct {

	/*
	  In: Body
	*/
	Payload *models.Price `json:"body,omitempty"`
}

// NewGetFeeVotesProducerUsdOK creates GetFeeVotesProducerUsdOK with default headers values
func NewGetFeeVotesProducerUsdOK() *GetFeeVotesProducerUsdOK {

	return &GetFeeVotesProducerUsdOK{}
}

// WithPayload adds the payload to the get fee votes producer usd o k response
func (o *GetFeeVotesProducerUsdOK) WithPayload(payload *models.Price) *GetFeeVotesProducerUsdOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get fee votes producer usd o k response
func (o *GetFeeVotesProducerUsdOK) SetPayload(payload *models.Price) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetFeeVotesProducerUsdOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// GetFeeVotesProducerUsdBadRequestCode is the HTTP code returned for type GetFeeVotesProducerUsdBadRequest
const GetFeeVotesProducerUsdBadRequestCode int = 400

/*GetFeeVotesProducerUsdBadRequest Invalid account format, should be a 12 character string

swagger:response getFeeVotesProducerUsdBadRequest
*/
type GetFeeVotesProducerUsdBadRequest struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewGetFeeVotesProducerUsdBadRequest creates GetFeeVotesProducerUsdBadRequest with default headers values
func NewGetFeeVotesProducerUsdBadRequest() *GetFeeVotesProducerUsdBadRequest {

	return &GetFeeVotesProducerUsdBadRequest{}
}

// WithPayload adds the payload to the get fee votes producer usd bad request response
func (o *GetFeeVotesProducerUsdBadRequest) WithPayload(payload *models.Error) *GetFeeVotesProducerUsdBadRequest {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get fee votes producer usd bad request response
func (o *GetFeeVotesProducerUsdBadRequest) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetFeeVotesProducerUsdBadRequest) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(400)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// GetFeeVotesProducerUsdNotFoundCode is the HTTP code returned for type GetFeeVotesProducerUsdNotFound
const GetFeeVotesProducerUsdNotFoundCode int = 404

/*GetFeeVotesProducerUsdNotFound Did not find a matching producer

swagger:response getFeeVotesProducerUsdNotFound
*/
type GetFeeVotesProducerUsdNotFound struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewGetFeeVotesProducerUsdNotFound creates GetFeeVotesProducerUsdNotFound with default headers values
func NewGetFeeVotesProducerUsdNotFound() *GetFeeVotesProducerUsdNotFound {

	return &GetFeeVotesProducerUsdNotFound{}
}

// WithPayload adds the payload to the get fee votes producer usd not found response
func (o *GetFeeVotesProducerUsdNotFound) WithPayload(payload *models.Error) *GetFeeVotesProducerUsdNotFound {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get fee votes producer usd not found response
func (o *GetFeeVotesProducerUsdNotFound) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetFeeVotesProducerUsdNotFound) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(404)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// GetFeeVotesProducerUsdServiceUnavailableCode is the HTTP code returned for type GetFeeVotesProducerUsdServiceUnavailable
const GetFeeVotesProducerUsdServiceUnavailableCode int = 503

/*GetFeeVotesProducerUsdServiceUnavailable Data is stale, has not been updated for several minutes

swagger:response getFeeVotesProducerUsdServiceUnavailable
*/
type GetFeeVotesProducerUsdServiceUnavailable struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewGetFeeVotesProducerUsdServiceUnavailable creates GetFeeVotesProducerUsdServiceUnavailable with default headers values
func NewGetFeeVotesProducerUsdServiceUnavailable() *GetFeeVotesProducerUsdServiceUnavailable {

	return &GetFeeVotesProducerUsdServiceUnavailable{}
}

// WithPayload adds the payload to the get fee votes producer usd service unavailable response
func (o *GetFeeVotesProducerUsdServiceUnavailable) WithPayload(payload *models.Error) *GetFeeVotesProducerUsdServiceUnavailable {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get fee votes producer usd service unavailable response
func (o *GetFeeVotesProducerUsdServiceUnavailable) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetFeeVotesProducerUsdServiceUnavailable) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(503)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}
